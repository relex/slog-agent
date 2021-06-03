package base

// MultiChannelBufferReceiver is the receiver for multi-channel input of log record buffers
type MultiChannelBufferReceiver interface {

	// NewChannel creates a receiver for a new channel identified by the given id
	// The id may be reused after the previous channel closes, e.g. a local TCP port
	NewChannel(id string) BufferReceiverChannel

	// Destroy performs cleanup
	Destroy()
}

// BufferReceiverChannel is under MultiChannelBufferReceiver to receive input from e.g. a single TCP connection
type BufferReceiverChannel interface {

	// Accept takes new buffer of log records from input
	// The buffer is NOT usable after the function exits
	Accept(buffer []*LogRecord)

	// Tick is called periodically for custom operations
	// Tick is guaranteed to be called periodically, though interval can vary and be inaccurate
	Tick()

	// Close is called when this channel is ended
	Close()
}
