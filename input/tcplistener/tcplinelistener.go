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

const tcpReadBufferMax = 8 * 1024 * 1024 // Less than /proc/sys/net/ipv4/tcp_mem
const tcpReadBufferMin = 65536

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
	receiver base.MultiSinkMessageReceiver, stopRequest channels.Awaitable) (base.LogListener, string, error) {

	// open TCP socket
	socket, err := net.Listen("tcp", address)
	if err != nil {
		return nil, "", err
	}
	boundAddr := socket.Addr().String()

	logger := parentLogger.WithFields(logger.Fields{
		defs.LabelComponent: "TCPLineListener",
		defs.LabelAddress:   boundAddr,
	})
	logger.Info("start listening")

	// init taskCounter with 1 for the listener; Can't wait for Start() because WaitGroupAwaitable below would quit immediately if it's zero.
	taskCounter := &sync.WaitGroup{}
	taskCounter.Add(1)

	return &tcpLineListener{
		logger:      logger,
		socket:      socket.(*net.TCPListener),
		testRecord:  testRecord,
		receiver:    receiver,
		stopRequest: stopRequest,
		stopTimeout: stopRequest.After(defs.IntermediateChannelTimeout),
		taskCounter: taskCounter,
		stopped:     channels.NewWaitGroupAwaitable(taskCounter), // input is only fully stopped after all connections are closed
	}, boundAddr, nil
}

func (lsnr *tcpLineListener) Start() {
	go lsnr.run()
}

func (lsnr *tcpLineListener) Stopped() channels.Awaitable {
	return lsnr.stopped
}

func (lsnr *tcpLineListener) run() {
	// background goroutine to wait and close listener on request
	abortListener := channels.NewSignalAwaitable()
	go func() {
		channels.AnyAwaitables(lsnr.stopRequest, abortListener).Next(func() {
			if abortListener.Peek() {
				lsnr.logger.Info("abort listener")
			} else {
				lsnr.logger.Info("close listener on stop request")
			}
		}).WaitForever()
		lsnr.socket.Close()
	}()

	// main loop
	lsnr.logger.Info("start accept loop")
	for {
		conn, err := lsnr.socket.AcceptTCP()
		if err != nil {
			if lsnr.stopRequest.Peek() && util.IsNetworkClosed(err) {
				// closed on stop request
			} else {
				lsnr.logger.Error("accept() error: ", err)
				abortListener.Signal()
			}
			break
		}

		clientNumber := base.ClientNumber(util.GetFDFromTCPConnOrPanic(conn))
		connLogger := lsnr.logger.WithFields(logger.Fields{
			defs.LabelPart:         "connection",
			defs.LabelClient:       conn.RemoteAddr().String(),
			defs.LabelClientNumber: clientNumber,
		})
		if clientNumber >= base.MaxClientNumber {
			connLogger.Error("rejected connection: too many clients")
			conn.Close()
			continue
		}

		connLogger.Info("accepted connection")
		lsnr.taskCounter.Add(1)
		go lsnr.runConnection(connLogger, conn, clientNumber)
	}
	lsnr.logger.Info("end accept loop")

	// mark the listener itself as done, note there could still be established connections
	lsnr.taskCounter.Done()
}

func (lsnr *tcpLineListener) runConnection(connLogger logger.Logger, conn *net.TCPConn, clientNumber base.ClientNumber) {
	defer lsnr.taskCounter.Done()
	connLogger.Info("started")

	recvChan := lsnr.receiver.NewSink(conn.RemoteAddr().String(), clientNumber)
	defer recvChan.Close()

	connAborter := lsnr.launchConnectionCloser(connLogger, conn)

	// short timeout for periodic flushing
	connReader := lsnr.createConnectionReader(connLogger, conn)
	mlineReader := newMultiLineReader(connReader.Read, lsnr.testRecord,
		defs.ListenerLineBufferSize, defs.InputLogMaxMessageBytes, recvChan.Accept)

	emptyTime := time.Time{}
	prevDeadline := time.Time{}
	for {
		err := mlineReader.Read()
		if err == nil {
			if prevDeadline == emptyTime {
				prevDeadline = connReader.ReadDeadline()
			} else if connReader.ReadDeadline() != prevDeadline {
				connLogger.Debug("flush input for deadline update")
				mlineReader.Flush()
				recvChan.Flush()
				prevDeadline = connReader.ReadDeadline()
			}
			continue
		}
		if util.IsNetworkTimeout(err) {
			connLogger.Debug("flush input for timeout")
			// TODO: close lingering connection
			mlineReader.Flush()
			recvChan.Flush()
			continue
		}
		// error handling
		mlineReader.FlushAll()
		if util.IsNetworkClosed(err) && lsnr.stopRequest.Peek() {
			// already closed by connAborter
			connLogger.Info("closed by stop request (delayed)")
		} else {
			if !util.IsNetworkClosed(err) {
				connLogger.Warn("read() error: ", err)
			}
			connAborter.Signal()
		}
		break
	}

	recvChan.Flush()
	connLogger.Info("ended")
}

func (lsnr *tcpLineListener) launchConnectionCloser(connLogger logger.Logger, conn *net.TCPConn) *channels.SignalAwaitable {
	abortConn := channels.NewSignalAwaitable()
	// background goroutine to wait and close listener on request
	go func() {
		channels.AnyAwaitables(lsnr.stopRequest, abortConn).Next(func() {
			if abortConn.Peek() {
				connLogger.Info("abort connection")
			} else {
				connLogger.Info("close connection on stop request")
			}
		}).WaitForever()
		conn.Close()
	}()
	return abortConn
}

func (lsnr *tcpLineListener) createConnectionReader(connLogger logger.Logger, conn *net.TCPConn) *util.NetConnWrapper {
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
