package fluentdforward

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/output/baseoutput"
)

// clientWorker is a client of fluentd Forward protocol to forward chunks to upstream
type clientWorker struct {
	logger       logger.Logger
	config       UpstreamConfig
	inputChannel <-chan base.LogChunk
	inputClosed  channels.Awaitable
	onChunkAcked func(chunk base.LogChunk)
	onChunkLeft  func(chunk base.LogChunk)
	onFinished   func()
	stopped      *channels.SignalAwaitable
	metrics      baseoutput.ClientMetrics
}

// NewClientWorker creates ClientWorker
func NewClientWorker(parentLogger logger.Logger, args base.ChunkConsumerArgs, config UpstreamConfig, metricFactory *base.MetricFactory) base.ChunkConsumer {
	client := &clientWorker{
		logger:       parentLogger.WithField(defs.LabelComponent, "FluentdForwardClient"),
		config:       config,
		inputChannel: args.InputChannel,
		inputClosed:  args.InputClosed,
		onChunkAcked: args.OnChunkConsumed,
		onChunkLeft:  args.OnChunkLeftover,
		onFinished:   args.OnFinished,
		stopped:      channels.NewSignalAwaitable(),
		metrics:      baseoutput.NewClientMetrics(metricFactory),
	}
	return client
}

// Launch starts the ClientWorker
func (client *clientWorker) Launch() {
	go client.run()
}

// Stopped returns an Awaitable which is signaled when stopped
func (client *clientWorker) Stopped() channels.Awaitable {
	return client.stopped
}

func (client *clientWorker) run() {
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

func (client *clientWorker) runConnection(leftovers chan base.LogChunk) (chan base.LogChunk, bool) {
	conn, oerr := client.openConnection()
	if oerr != nil {
		client.logger.Warnf("failed to open connection: %s", oerr.Error())
		client.metrics.IncrementNetworkErrors()
		return leftovers, true
	}
	defer func() {
		client.logger.Infof("close connection")
		conn.Close() // ignore error because Close may have been called by acknowledger
	}()
	client.logger.Infof("handshaking with %s", conn.RemoteAddr())
	success, reason, herr := forwardprotocol.DoClientHandshake(conn, client.config.Secret, defs.ForwarderHandshakeTimeout)
	if herr != nil {
		client.logger.Warnf("handshake failed due to network error: %s", herr.Error())
		client.metrics.IncrementNetworkErrors()
		return leftovers, true
	} else if !success {
		client.logger.Errorf("handshake failed due to misconfiguration: %s", reason)
		client.metrics.IncrementNonNetworkErrors()
		return leftovers, true
	}
	session := newClientSession(client, conn, leftovers)
	return session.run()
}

func (client *clientWorker) openConnection() (net.Conn, error) {
	var conn net.Conn
	var err error
	if client.config.TLS {
		client.logger.Infof("connecting to %s in TLS mode", client.config.Address)
		dialer := &net.Dialer{}
		dialer.Timeout = defs.ForwarderConnectionTimeout
		dialer.Deadline = time.Now().Add(defs.ForwarderConnectionTimeout)
		tlsConfig := &tls.Config{}
		tlsConfig.InsecureSkipVerify = true
		conn, err = tls.DialWithDialer(dialer, "tcp", client.config.Address, tlsConfig)
	} else {
		client.logger.Infof("connecting to %s in TCP mode", client.config.Address)
		conn, err = net.DialTimeout("tcp", client.config.Address, defs.ForwarderConnectionTimeout)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}
