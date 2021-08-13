package baseoutput

import (
	"time"

	"github.com/relex/slog-agent/base"
)

// EstablishConnectionFunc opens a ClientConnection to upstream
type EstablishConnectionFunc func() (ClientConnection, error)

// ClientConnection represents a connection / session / channel to upstream.
//
// It must support full-duplex or desynchronized input and output to allow pipelining, i.e. send next chunk without
// waiting for the current chunk to finish
type ClientConnection interface {
	RemoteAddr() string

	// SendChunk sends out the given chunk to remote
	SendChunk(chunk base.LogChunk, deadline time.Time) error

	// SendPing sends a ping singal every N seconds, depending on the caller
	//
	// If ping is not supported by the underlying protocol, the function should do nothing
	SendPing(deadline time.Time) error

	// ReadChunkAck reads an ID of an acknowledged chunk
	//
	// Acknowledgement of chunks by upstream is not required to be in the same order they were sent.
	//
	// If the order is maintained, it may return empty string to simply indicate the next chunk.
	ReadChunkAck(deadline time.Time) (string, error)

	// Close closes the connection
	//
	// Close may be called multiple times and it's not considered an error.
	Close()
}
