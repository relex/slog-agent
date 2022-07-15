package fluentdforward

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/output/baseoutput"
	"github.com/vmihailenco/msgpack/v4"
)

type forwardConnection struct {
	logger  logger.Logger
	socket  net.Conn
	decoder msgpack.Decoder // to read msgpack responses from Fluentd
}

var internalPingMessage = buildInternalPingMessage()

// NewClientWorker creates ClientWorker
func NewClientWorker(parentLogger logger.Logger, args base.ChunkConsumerArgs, config UpstreamConfig, metricCreator promreg.MetricCreator) base.ChunkConsumer {
	clientLogger := parentLogger.WithField(defs.LabelComponent, "FluentdForwardClient")

	return baseoutput.NewClientWorker(
		clientLogger,
		args,
		metricCreator,
		func() (baseoutput.ClosableClientConnection, error) {
			return openForwardConnection(clientLogger, config)
		},
		config.MaxDuration,
		"fluentd",
	)
}

func openForwardConnection(parentLogger logger.Logger, config UpstreamConfig) (baseoutput.ClosableClientConnection, error) {
	connLogger := parentLogger.WithField(defs.LabelServer, config.Address)

	sock, sockErr := connect(connLogger, config.TLS, config.Address)
	if sockErr != nil {
		return nil, fmt.Errorf("failed to connect: %w", sockErr)
	}
	connLogger.Info("connected to ", sock.RemoteAddr())

	success, reason, herr := forwardprotocol.DoClientHandshake(sock, config.Secret, defs.ForwarderHandshakeTimeout)
	if herr != nil {
		sock.Close()
		return nil, fmt.Errorf("failed to handshake: %w", herr)
	}
	if !success {
		sock.Close()
		return nil, fmt.Errorf("failed to handshake: %s", reason)
	}

	return &forwardConnection{
		logger:  connLogger,
		socket:  sock,
		decoder: *msgpack.NewDecoder(sock),
	}, nil
}

func connect(connLogger logger.Logger, useTLS bool, address string) (net.Conn, error) {
	var sock net.Conn
	var err error

	if useTLS {
		connLogger.Infof("connecting to %s in TLS mode", address)
		dialer := &net.Dialer{}
		dialer.Timeout = defs.ForwarderConnectionTimeout
		dialer.Deadline = time.Now().Add(defs.ForwarderConnectionTimeout)
		tlsConfig := &tls.Config{}
		tlsConfig.InsecureSkipVerify = true
		sock, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		connLogger.Infof("connecting to %s in TCP mode", address)
		sock, err = net.DialTimeout("tcp", address, defs.ForwarderConnectionTimeout)
	}

	if err != nil {
		return nil, err
	}
	return sock, nil
}

func (fconn *forwardConnection) Logger() logger.Logger {
	return fconn.logger
}

func (fconn *forwardConnection) SendChunk(chunk base.LogChunk, deadline time.Time) error {
	if err := fconn.socket.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("failed to set send timeout: %s, %w", chunk.String(), err)
	}

	if err := writeAll(fconn.socket, chunk.Data); err != nil {
		return fmt.Errorf("failed to send: %s, %w", chunk.String(), err)
	}

	return nil
}

func (fconn *forwardConnection) SendPing(deadline time.Time) error {
	if err := fconn.socket.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("failed to set ping timeout: %w", err)
	}

	if err := writeAll(fconn.socket, internalPingMessage); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	return nil
}

func (fconn *forwardConnection) ReadChunkAck(deadline time.Time) (string, error) {
	if err := fconn.socket.SetReadDeadline(deadline); err != nil {
		return "", fmt.Errorf("failed to set read timeout: %w", err)
	}

	ack := forwardprotocol.Ack{}
	if err := fconn.decoder.Decode(&ack); err != nil {
		return "", fmt.Errorf("failed to read ACK: %w", err)
	}

	return ack.Ack, nil
}

func (fconn *forwardConnection) Close() {
	fconn.socket.Close()
}

func buildInternalPingMessage() []byte {
	var packet bytes.Buffer
	packet.Grow(100)
	encoder := msgpack.NewEncoder(&packet)
	// root array
	if err := encoder.EncodeArrayLen(3); err != nil {
		logger.Panic(err)
	}
	{
		// root[0]: tag
		if err := encoder.EncodeString("internal.ping"); err != nil {
			logger.Panic(err)
		}
		// root[1]: array of log records in batch
		if err := encoder.EncodeArrayLen(0); err != nil {
			logger.Panic(err)
		}
		// root[2]: options
		if err := encoder.Encode(forwardprotocol.TransportOption{
			Size:       0,
			Chunk:      "",
			Compressed: "",
		}); err != nil {
			logger.Panic(err)
		}
	}
	return packet.Bytes()
}

func writeAll(conn io.Writer, data []byte) error {
	for {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
		if len(data) == 0 {
			return nil
		}
	}
}
