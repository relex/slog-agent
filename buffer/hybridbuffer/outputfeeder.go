package hybridbuffer

import (
	"sync"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

type outputFeeder struct {
	logger          logger.Logger
	chunkMan        chunkManager
	consumerCounter *sync.WaitGroup
	inputChannel    <-chan base.LogChunk // internal; LogChunk.Data can be nil if unloaded / saved on disk
	inputClosed     channels.Awaitable   // internal; to abort ongoing input processing
	metrics         bufferMetrics
	outputChannel   chan base.LogChunk        // normal LogChunk
	outputClosed    *channels.SignalAwaitable // to abort output processing if consumers are not waiting on output
	stopped         *channels.SignalAwaitable
}

func newOutputFeeder(parentLogger logger.Logger, chunkMan chunkManager,
	inputChannel <-chan base.LogChunk, inputClosed channels.Awaitable, metrics bufferMetrics,
) outputFeeder {
	flogger := parentLogger.WithField(defs.LabelPart, "OutputFeeder")
	return outputFeeder{
		logger:          flogger,
		chunkMan:        chunkMan,
		consumerCounter: &sync.WaitGroup{},
		inputChannel:    inputChannel,
		inputClosed:     inputClosed,
		metrics:         metrics,
		outputChannel:   make(chan base.LogChunk, defs.BufferMaxNumChunksInMemory),
		outputClosed:    channels.NewSignalAwaitable(),
		stopped:         channels.NewSignalAwaitable(),
	}
}

// RegisterNewConsumer creates the parameters for a new consumer.
// The args must be used by a newly launched consumer and call OnFinished at the end.
func (feeder *outputFeeder) RegisterNewConsumer() base.ChunkConsumerArgs {
	feeder.consumerCounter.Add(1)
	return base.ChunkConsumerArgs{
		InputChannel:    feeder.outputChannel,
		InputClosed:     feeder.outputClosed,
		OnChunkConsumed: feeder.chunkMan.OnChunkConsumed,
		OnChunkLeftover: feeder.chunkMan.OnChunkLeftover,
		OnFinished:      feeder.consumerCounter.Done,
	}
}

func (feeder *outputFeeder) NumOutput() int {
	return len(feeder.outputChannel)
}

func (feeder *outputFeeder) Stopped() channels.Awaitable {
	return feeder.stopped
}

func (feeder *outputFeeder) Run() {
	feeder.logger.Info("start main loop")

	// pass chunks from input channel (maybe unloaded) to output channel (fully loaded)
	var lastInputChunk base.LogChunk
	for {
		chunk, ok := <-feeder.inputChannel // wait forever here
		if !ok {
			feeder.logger.Infof("end main loop on input channel close, remaining=%d", len(feeder.inputChannel))
			break
		}

		if chunk.Data != nil {
			feeder.metrics.queuedChunksTransient.Dec()
		} else {
			feeder.metrics.queuedChunksPersistent.Dec()
		}

		if !feeder.loadToOutput(chunk) {
			feeder.logger.Infof("end main loop on output stop signal, remaining=%d", len(feeder.inputChannel))
			lastInputChunk = chunk
			break
		}
	}

	// clean up
	close(feeder.outputChannel)
	feeder.outputClosed.Signal()
	feeder.saveEverything(lastInputChunk)

	// wait for consumers here because the callbacks depend on chunkMan/dir
	feeder.logger.Infof("waiting for consumers: count=%d", util.PeekWaitGroup(feeder.consumerCounter))
	feeder.consumerCounter.Wait()
	feeder.chunkMan.Close()
	feeder.stopped.Signal()
	feeder.logger.Info("ended")
}

func (feeder *outputFeeder) loadToOutput(chunk base.LogChunk) bool {
	feeder.logger.Debugf("load chunk from queue: id=%s saved=%t", chunk.ID, chunk.Saved)
	if !feeder.chunkMan.LoadOrDropChunk(&chunk) {
		return true
	}

	select {
	case feeder.outputChannel <- chunk: // wait forever here
		return true
	case <-feeder.inputClosed.Channel():
		return false
	}
}

func (feeder *outputFeeder) saveEverything(lastInputChunk base.LogChunk) {
	numSaved := 0
	numDropped := 0

	// try to save all chunks in inputChannel
	for chunk := range feeder.inputChannel {
		// scopelint:ignore
		if feeder.chunkMan.UnloadOrDropChunk(&chunk) {
			numSaved++
		} else {
			numDropped++
		}
	}

	// try to save the unoutputted chunk from main loop
	if lastInputChunk.ID != "" {
		if feeder.chunkMan.UnloadOrDropChunk(&lastInputChunk) {
			numSaved++
		} else {
			numDropped++
		}
	}

	// try to save all chunks in outputChannel (consumers already quit)
	for chunk := range feeder.outputChannel {
		// scopelint:ignore
		if feeder.chunkMan.UnloadOrDropChunk(&chunk) {
			numSaved++
		} else {
			numDropped++
		}
	}

	feeder.logger.Infof("cleanup complete: saved=%d dropped=%d", numSaved, numDropped)
}
