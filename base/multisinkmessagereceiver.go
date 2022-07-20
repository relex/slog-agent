package base

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/defs"
)

// ClientNumber uniquely identifies a currently connected client from one of the inputs
//
// It's currently set to the FD of incoming connections
//
// TODO: use something other than FD in future for inputs not based on FD
type ClientNumber uint

// MaxClientNumber represents the max value of client number passed to MultiSinkMessageReceiver.NewSink
//
// The value affects how many clients we can have at the same time. It should be theorically unreachable.
//
// The value must be a constant for fix-sized arrays to be declared.
const MaxClientNumber ClientNumber = 262144

// MultiSinkMessageReceiver receives raw log messages from a multi-source input, e.g. a TCP listener with different incoming connections
//
// For an ordinary TCP input, there is a single MultiSinkMessageReceiver, and one MessageReceiverSink for each connection
//
// The only production implementation is LogParsingReceiver
type MultiSinkMessageReceiver interface {

	// NewSink creates a sink to receive raw messages from the input source identified by the given id or client number
	//
	// clientAddress is a descriptive string of address, e.g. "10.1.0.1:50001"
	//
	// clientNumber identifies a currently connected client from one of the inputs, e.g. socket FD.
	NewSink(clientAddress string, clientNumber ClientNumber) MessageReceiverSink
}

// MessageReceiverSink receives raw log messages from a single source, e.g. a client TCP connection
type MessageReceiverSink interface {

	// Accept receives a raw message from input
	//
	// The message slice is NOT usable after the function exits
	Accept(message []byte)

	// Flush is called periodically for custom operations
	//
	// Flush is guaranteed to be called periodically, though interval can vary and be inaccurate
	//
	// Flush is also to be called right before Close(), after all Accept() calls
	Flush()

	// Close is called when the source feeding this sink is ended
	Close()
}

// NewSinkLogger creates a derived logger for sinks created from MultiSinkMessageReceiver or MultiSinkBufferReceiver
func NewSinkLogger(parentLogger logger.Logger, clientAddress string, clientNumber ClientNumber) logger.Logger {
	return parentLogger.WithFields(logger.Fields{
		defs.LabelPart:         "sink",
		defs.LabelClient:       clientAddress,
		defs.LabelClientNumber: clientNumber,
	})
}
