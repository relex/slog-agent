package baseoutput

import (
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// ClientWorker is a common client implementing ChunkConsumer
//
// The caller provides a minimum ClientConnection through EstablishConnectionFunc, while ClientWorker handles
// logging, metrics, error recovery, reconnecting, periodic ping, and pipelining by handling sending and receiving on
// separate goroutines
type ClientWorker struct {
	logger        logger.Logger
	inputChannel  <-chan base.LogChunk
	inputClosed   channels.Awaitable
	onChunkAcked  func(chunk base.LogChunk)
	onChunkLeft   func(chunk base.LogChunk)
	onFinished    func()
	stopped       *channels.SignalAwaitable
	metrics       clientMetrics
	openConn      EstablishConnectionFunc
	maxDuration   time.Duration                 // max duration of session before reconnection
	activeSession util.AtomicRef[clientSession] // holding place of the current clientSession
}

// NewClientWorker creates ClientWorker
func NewClientWorker(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricCreator promreg.MetricCreator,
	openConn EstablishConnectionFunc, maxDuration time.Duration) base.ChunkConsumer {

	client := &ClientWorker{
		logger:        parentLogger,
		inputChannel:  args.InputChannel,
		inputClosed:   args.InputClosed,
		onChunkAcked:  args.OnChunkConsumed,
		onChunkLeft:   args.OnChunkLeftover,
		onFinished:    args.OnFinished,
		stopped:       channels.NewSignalAwaitable(),
		metrics:       newClientMetrics(metricCreator),
		openConn:      openConn,
		maxDuration:   maxDuration,
		activeSession: util.AtomicRef[clientSession]{},
	}

	// fast shutdown: force immediate ending of output when inputClosed is signaled
	// the closing should abort any ongoing network I/O operations
	//
	// This is designed for busy clients as they cannot allow logging being paused for extended period of time
	// (most have no queue and logging is usually a blocking operation), at the cost of resending pending logs which
	// would cause duplications.
	//
	// TODO: make this an option or dependent on keys/tags
	client.inputClosed.Next(func() {
		sess := client.activeSession.Get()
		if sess != nil {
			sess.Abort(func() {
				client.logger.Info("abort ongoing connection due to stop request")
			})
		}
	})

	return client
}

// Start starts the ClientWorker
func (client *ClientWorker) Start() {
	go client.run()
}

// Stopped returns an Awaitable which is signaled when stopped
func (client *ClientWorker) Stopped() channels.Awaitable {
	return client.stopped
}

func (client *ClientWorker) run() {
	defer client.stopped.Signal()
	defer client.onFinished()
	leftovers := make(chan base.LogChunk)
	client.logger.Infof("started")

	func() {
		for {
			var retry reconnectPolicy
			leftovers, retry = client.runSession(leftovers)
			switch retry {
			case noReconnect:
				client.logger.Infof("stop requested (session)")
				return
			case reconnectWithDelay:
				if client.inputClosed.Wait(defs.ForwarderRetryInterval) { // false if timeout, which is expected
					client.logger.Infof("stop requested (retry wait)")
					return
				}
			case reconnect:
				client.logger.Info("reconnect without delay requested for load balancing")
			default:
				client.logger.Panic("BUG: unexpected reconnectPolicy: ", retry)
			}

			client.logger.Infof("retrying connection, leftovers=%d", len(leftovers))
		}
	}()

	close(leftovers)
	client.logger.Infof("save on shutdown, leftovers=%d", len(leftovers))
	for chunk := range leftovers {
		client.metrics.OnLeftoverPopped(chunk)
		client.onChunkLeft(chunk)
	}
	client.logger.Info("stopped")
}

func (client *ClientWorker) runSession(leftovers chan base.LogChunk) (chan base.LogChunk, reconnectPolicy) {

	// open the connection in the background so we can abort anytime by signal
	connCh := make(chan ClosableClientConnection)
	go func() {
		conn, err := client.openConn()
		if err != nil {
			client.logger.Warnf("failed to open connection: %s", err.Error())
			client.metrics.OnError(err)
			// send nil to signal an error
			connCh <- nil
			return
		}
		// send the newly created connection back
		connCh <- conn
	}()

	var conn ClosableClientConnection
	// wait for above goroutine to end OR inputClosed signal
	select {
	case <-client.inputClosed.Channel():
		client.logger.Info("stop requested (connection opening stage)")
		return leftovers, noReconnect // ignore the connection being opened in background as this is full shutdown
	case conn = <-connCh:
		if conn == nil {
			// the connection opening failed
			return leftovers, reconnectWithDelay
		}
		// continue after connection opening
	}

	client.metrics.OnOpening()

	sess := newClientSession(client, conn)
	client.activeSession.Set(sess)

	defer func() {
		sess.Abort(func() {
			client.logger.Info("close connection at the end of session")
		})
		client.activeSession.Set(nil)
	}()

	return sess.Run(leftovers, client.maxDuration)
}
