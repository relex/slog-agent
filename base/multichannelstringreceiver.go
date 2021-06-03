package base

// MultiChannelStringReceiver is the receiver for multi-channel input, e.g. TCP listener with indepndent incoming connections
type MultiChannelStringReceiver interface {

	// NewChannel creates a receiver for a new channel identified by the given id
	// The id may be reused after the previous channel closes
	NewChannel(id string) StringReceiverChannel

	// Destroy performs cleanup
	Destroy()
}

// StringReceiverChannel is under MultiChannelStringReceiver to receive input from e.g. a single TCP connection
type StringReceiverChannel interface {

	// Accept takes new data from input
	// The value is NOT usable after the function exits
	Accept(value []byte)

	// Flush is called periodically for custom operations
	// Flush is guaranteed to be called periodically, though interval can vary and be inaccurate
	// Flush is also to be called right before Close(), after all Accept() calls
	Flush()

	// Close is called when this channel is ended
	Close()
}
