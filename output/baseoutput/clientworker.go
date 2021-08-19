package baseoutput

import (
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// ClientWorker is a common client implementing ChunkConsumer
//
// The caller provides a minimum ClientConnection through EstablishConnectionFunc, while ClientWorker handles
// logging, metrics, error recovery, reconnecting, periodical ping, and pipeling by handling sending and receiving on
// their respective goroutines
type ClientWorker struct {
	logger       logger.Logger
	inputChannel <-chan base.LogChunk
	inputClosed  channels.Awaitable
	onChunkAcked func(chunk base.LogChunk)
	onChunkLeft  func(chunk base.LogChunk)
	onFinished   func()
	stopped      *channels.SignalAwaitable
	metrics      clientMetrics
	openConn     EstablishConnectionFunc
}

// NewClientWorker creates ClientWorker
func NewClientWorker(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricFactory *base.MetricFactory,
	openConn EstablishConnectionFunc) base.ChunkConsumer {

	client := &ClientWorker{
		logger:       parentLogger,
		inputChannel: args.InputChannel,
		inputClosed:  args.InputClosed,
		onChunkAcked: args.OnChunkConsumed,
		onChunkLeft:  args.OnChunkLeftover,
		onFinished:   args.OnFinished,
		stopped:      channels.NewSignalAwaitable(),
		metrics:      newClientMetrics(metricFactory),
		openConn:     openConn,
	}
	return client
}

// Launch starts the ClientWorker
func (client *ClientWorker) Launch() {
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
	for {
		var retry bool
		if leftovers, retry = client.runConnection(leftovers); !retry {
			client.logger.Infof("stop requested (connection)")
			break
		}
		if client.inputClosed.Wait(defs.ForwarderRetryInterval) { // false if timeout, which is expected
			client.logger.Infof("stop requested (retry wait)")
			break
		}
		client.logger.Infof("retrying connection with leftovers=%d", len(leftovers))
	}
	close(leftovers)
	client.logger.Infof("process leftovers=%d on shutdown", len(leftovers))
	for chunk := range leftovers {
		client.metrics.OnLeftoverPopped(chunk)
		client.onChunkLeft(chunk)
	}
	client.logger.Info("stopped")
}

func (client *ClientWorker) runConnection(leftovers chan base.LogChunk) (chan base.LogChunk, bool) {
	conn, err := client.openConn()
	if err != nil {
		client.logger.Warnf("failed to open connection: %s", err.Error())
		client.metrics.OnError(err)
		return leftovers, true
	}

	defer func() {
		client.logger.Infof("close connection")
		conn.Close() // ignore error because Close may have been called by acknowledger
	}()

	return newClientSession(client, conn, leftovers).run()
}
