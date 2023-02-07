package base

// ChunkBufferer is a worker to buffer completed chunks in memory and/or persistent storage
// Accept can be called concurrently by different goroutines
// It is to the bufferer to process any chunks left in input or output channels during shutdown
type ChunkBufferer interface {
	PipelineWorker

	// RegisterNewConsumer registers a new consumer to be launched and returns args for its construction
	RegisterNewConsumer() ChunkConsumerArgs

	// Accept accepts incoming log chunks
	Accept(chunk LogChunk)

	// Destroy destroys the buffer, saves all remaining chunks in channels and waiting for consumers to finish
	Destroy()
}
