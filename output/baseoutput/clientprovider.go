package baseoutput

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

// EstablishConnectionFunc opens a ClosableClientConnection to upstream
type EstablishConnectionFunc func() (ClosableClientConnection, error)

// ClientConnection represents a connection / session / channel to upstream.
//
// It must support full-duplex or desynchronized input and output to allow pipelining, i.e. send next chunks without
// waiting for the current chunk to be acknowledged.
type ClientConnection interface {

	// Logger returns the logger bound to this connection
	Logger() logger.Logger

	// SendChunk sends out the given chunk to remote destination
	SendChunk(chunk base.LogChunk, deadline time.Time) error

	// SendPing sends a ping signal.
	//
	// If ping is not supported by the underlying protocol, the function should do nothing
	SendPing(deadline time.Time) error

	// ReadChunkAck reads the ID of a previously-sent chunk once it's acknowledged.
	//
	// Acknowledgement of chunks by upstream is not required to be in the same order they were sent.
	//
	// If the order is strictly maintained, it may return empty string to indicate the last unacknowledged chunk.
	ReadChunkAck(deadline time.Time) (string, error)
}

// ClosableClientConnection is ClientConnection with Close method
type ClosableClientConnection interface {
	ClientConnection

	// Close closes the connection
	//
	// Close may be called simultaneously while sending or reading is still in progress, and it must be able to cancel
	// ongoing operations immediately.
	Close()
}
