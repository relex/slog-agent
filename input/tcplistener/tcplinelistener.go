package tcplistener

import (
	"net"
	"sync"
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

const (
	tcpReadBufferMax = 8 * 1024 * 1024 // Less than /proc/sys/net/ipv4/tcp_mem
	tcpReadBufferMin = 65536
)

var tcpLastReadBufferSize = tcpReadBufferMax // shared for all connections. No need to sync access as it's just a cached number.

// tcpLineListener is a TCP Listener for line-based, request-only text protocol, with support for multi-line messages.
//
// The listener sends incoming messages into MultiSinkMessageReceiver.
//
// - Incoming bytes are buffered until a line can be recognized / with newline.
//
// - Incoming lines are buffered until the latest line can be recognized as the start of another message, or passes certain timeout.
//
// - The resulting messages don't contain newlines at the end, but can have newlines in the middle for multi-line messages.
//
// There is no request confirmation and the protocol is inheritantly unreliable.
type tcpLineListener struct {
	logger      logger.Logger
	socket      *net.TCPListener
	testRecord  func(ln []byte) bool
	receiver    base.MultiSinkMessageReceiver
	stopRequest channels.Awaitable
	stopTimeout channels.Awaitable // used by receiver; signaled X seconds after stopRequest to force shutdown
	taskCounter *sync.WaitGroup    // counter to track connection tasks and the listener task itself
	stopped     channels.Awaitable // stopped is signaled when both listener and all child connections have come to stop
}

// NewTCPLineListener creates a socket listening on the given TCP address and returns a new tcpLineListener if successful
//
// The given address may use port zero, which would cause the port to be assigned by OS
//
// Returns the listener, actual address including final port, and error if failed
func NewTCPLineListener(parentLogger logger.Logger, address string, testRecord func(ln []byte) bool,
	receiver base.MultiSinkMessageReceiver, stopRequest channels.Awaitable,
) (base.LogListener, string, error) {
	// open TCP socket
	socket, err := net.Listen("tcp", address)
	if err != nil {
		return nil, "", err
	}
	boundAddr := socket.Addr().String()

	log := parentLogger.WithFields(logger.Fields{
		defs.LabelComponent: "TCPLineListener",
		defs.LabelAddress:   boundAddr,
	})
	log.Info("start listening")

	// init taskCounter with 1 for the listener; Can't wait for Start() because WaitGroupAwaitable below would quit immediately if it's zero.
	taskCounter := &sync.WaitGroup{}
	taskCounter.Add(1)

	return &tcpLineListener{
		logger:      log,
		socket:      socket.(*net.TCPListener),
		testRecord:  testRecord,
		receiver:    receiver,
		stopRequest: stopRequest,
		stopTimeout: stopRequest.After(defs.IntermediateChannelTimeout),
		taskCounter: taskCounter,
		stopped:     channels.NewWaitGroupAwaitable(taskCounter), // input is only fully stopped after all connections are closed
	}, boundAddr, nil
}

func (listener *tcpLineListener) Start() {
	go listener.run()
}

func (listener *tcpLineListener) Stopped() channels.Awaitable {
	return listener.stopped
}

func (listener *tcpLineListener) run() {
	// background goroutine to wait and close listener on request
	abortListener := channels.NewSignalAwaitable()
	go func() {
		channels.AnyAwaitables(listener.stopRequest, abortListener).Next(func() {
			if abortListener.Peek() {
				listener.logger.Info("abort listener")
			} else {
				listener.logger.Info("close listener on stop request")
			}
		}).WaitForever()
		if err := listener.socket.Close(); err != nil {
			listener.logger.Warn("error closing listener: ", err)
		}
	}()

	// main loop
	listener.logger.Info("start accept() loop")
	for {
		newConn, acceptErr := listener.socket.AcceptTCP()
		if acceptErr != nil {
			if !(listener.stopRequest.Peek() && util.IsNetworkClosed(acceptErr)) {
				// not closed on stop request
				listener.logger.Warn("failed to accept() connection while listener is alive: ", acceptErr)
				abortListener.Signal()
			}
			break
		}

		newClientNumber := base.ClientNumber(util.GetFDFromTCPConnOrPanic(newConn))
		newConnLogger := listener.logger.WithFields(logger.Fields{
			defs.LabelPart:         "connection",
			defs.LabelClient:       newConn.RemoteAddr().String(),
			defs.LabelClientNumber: newClientNumber,
		})
		if newClientNumber >= base.MaxClientNumber {
			newConnLogger.Error("rejected new connection: too many clients")
			if err := newConn.Close(); err != nil { // we don't expect the client to close here
				listener.logger.Warn("error closing connection: ", err)
			}
			continue
		}

		newConnLogger.Info("accepted connection")
		listener.taskCounter.Add(1)
		go listener.runConnection(newConnLogger, newConn, newClientNumber)
	}
	listener.logger.Info("end accept() loop")

	// mark the listener itself as done, note there could still be established connections
	listener.taskCounter.Done()
}

func (listener *tcpLineListener) runConnection(connLogger logger.Logger, conn *net.TCPConn, clientNumber base.ClientNumber) {
	defer listener.taskCounter.Done()
	connLogger.Info("started")

	recvChan := listener.receiver.NewSink(conn.RemoteAddr().String(), clientNumber)
	defer recvChan.Close()

	connAborter := listener.launchConnectionCloser(connLogger, conn)

	// short timeout for periodic flushing
	connReader := listener.createConnectionReader(connLogger, conn)
	mlineReader := newMultiLineReader(connReader.Read, listener.testRecord,
		defs.ListenerLineBufferSize, defs.InputLogMaxTotalBytes, recvChan.Accept)

	emptyTime := time.Time{}
	prevDeadline := time.Time{}
	for {
		readErr := mlineReader.Read()
		if readErr == nil {
			if prevDeadline.Equal(emptyTime) {
				prevDeadline = connReader.ReadDeadline()
			} else if !connReader.ReadDeadline().Equal(prevDeadline) {
				connLogger.Debug("flush input for deadline update")
				mlineReader.Flush()
				recvChan.Flush()
				prevDeadline = connReader.ReadDeadline()
			}
			continue
		}

		// check if the short timeout (not real timeout) is reached and then flush buffer
		if util.IsNetworkTimeout(readErr) {
			connLogger.Debug("flush input")
			// TODO: close lingering connections if they don't send anything for hours or days
			mlineReader.Flush()
			recvChan.Flush()
			continue
		}
		
		// error handling
		mlineReader.FlushAll()
		if util.IsNetworkClosed(readErr) && listener.stopRequest.Peek() {
			// already closed by connAborter
			connLogger.Info("closed by stop request (delayed)")
		} else {
			if !util.IsNetworkClosed(readErr) {
				connLogger.Warn("read() error: ", readErr)
			}
			connAborter.Signal()
		}
		break
	}

	recvChan.Flush()
	connLogger.Info("ended")
}

func (listener *tcpLineListener) launchConnectionCloser(connLogger logger.Logger, conn *net.TCPConn) *channels.SignalAwaitable {
	abortConn := channels.NewSignalAwaitable()
	// background goroutine to wait and close listener on request
	go func() {
		channels.AnyAwaitables(listener.stopRequest, abortConn).Next(func() {
			if abortConn.Peek() {
				connLogger.Info("abort connection")
			} else {
				connLogger.Info("close connection on stop request")
			}
		}).WaitForever()
		if err := conn.Close(); err != nil && !util.IsNetworkClosed(err) {
			connLogger.Warn("error closing connection: ", err)
		}
	}()
	return abortConn
}

//nolint:revive
func (listener *tcpLineListener) createConnectionReader(connLogger logger.Logger, conn *net.TCPConn) *util.NetConnWrapper {
	if err := conn.SetKeepAlive(true); err != nil {
		connLogger.Warnf("error enabling keep-alive: %s", err.Error())
	}

	if sz, err := util.TrySetTCPReadBuffer(conn, tcpLastReadBufferSize, tcpReadBufferMin); err != nil {
		connLogger.Warnf("error changing buffer size: %s", err.Error())
	} else {
		connLogger.Infof("set TCP buffer size: %d", sz)
		tcpLastReadBufferSize = sz
	}

	return util.WrapNetConn(conn, defs.InputFlushInterval, 0)
}
