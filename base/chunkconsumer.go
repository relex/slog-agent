package base

import (
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
)

// ChunkConsumerConstructor creates a ChunkConsumer
type ChunkConsumerConstructor func(parentLogger logger.Logger, args ChunkConsumerArgs) ChunkConsumer

// ChunkConsumer is a worker to consume buffered chunks for forwarding or else
// A consumer should be created with ChunkConsumerArgs as input
// It should initiate shutdown by the end of InputChannel or the InputClosed signal,
// and never attempt to read any leftover chunk from InputChannel once it's closed
type ChunkConsumer interface {
	PipelineWorker
}

// ChunkConsumerArgs is the parameters to create a ChunkConsumer
// For any chunk, either OnChunkConsumed or OnChunkSkipped must be called
type ChunkConsumerArgs struct {
	InputChannel    <-chan LogChunk      // channel of fully loaded chunks to consume
	InputClosed     channels.Awaitable   // signal when input channel is closed, in case consumer is not waiting on input
	OnChunkConsumed func(chunk LogChunk) // to be called when a chunk is consumed / committed
	OnChunkLeftover func(chunk LogChunk) // to be called when a chunk is left unconsumed at the end
	OnFinished      func()               // to be called after the consumer ends
}
