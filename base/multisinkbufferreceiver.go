package base

// MultiSinkBufferReceiver receives buffered logs from a multi-source input, e.g. a TCP with different incoming connections
//
// For an ordinary TCP input, there is a single MultiSinkBufferReceiver, and one BufferReceiverSink for each connection
//
// The only production implementations are Orchestrator(s)
type MultiSinkBufferReceiver interface {
	// NewSink creates a sink to receive logs from the input identified by the given id
	//
	// clientAddress is a descriptive string of address, e.g. "10.1.0.1:50001"
	//
	// clientNumber identifies a currently connected client from one of the inputs, e.g. socket FD.
	NewSink(clientAddress string, clientNumber ClientNumber) BufferReceiverSink
}

// BufferReceiverSink under MultiSinkBufferReceiver receives buffered logs from a single source, e.g. a client TCP connection
type BufferReceiverSink interface {
	// Accept receives a buffer of log records from input
	//
	// The buffer is NOT usable after the function exits
	Accept(buffer []*LogRecord)

	// Tick is called periodically for custom operations
	//
	// Tick is guaranteed to be called periodically, though interval can vary and be inaccurate
	Tick()

	// Close is called when the source feeding this sink is ended
	Close()
}
