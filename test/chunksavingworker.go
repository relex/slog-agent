package test

import (
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// chunkSavingWorker collects processed logs for worker/agent-based benchmarks and integration tests
//
// chunkSavingWorker reuses chunkSaver internally
type chunkSavingWorker struct {
	logger       logger.Logger
	decoder      base.ChunkDecoder
	consumerArgs base.ChunkConsumerArgs
	saver        chunkSaver
	stopped      *channels.SignalAwaitable
}

func newChunkSavingWorker(parentLogger logger.Logger, chunkDecoder base.ChunkDecoder, args base.ChunkConsumerArgs,
	saver chunkSaver,
) base.ChunkConsumer {
	return &chunkSavingWorker{
		logger:       parentLogger.WithField(defs.LabelComponent, "ChunkSavingWorker"),
		decoder:      chunkDecoder,
		consumerArgs: args,
		saver:        saver,
		stopped:      channels.NewSignalAwaitable(),
	}
}

func (sworker *chunkSavingWorker) Start() {
	go sworker.run()
}

func (sworker *chunkSavingWorker) Stopped() channels.Awaitable {
	return sworker.stopped
}

func (sworker *chunkSavingWorker) run() {
	defer sworker.consumerArgs.OnFinished()
	defer sworker.stopped.Signal()
	defer sworker.saver.Close()
	ch := sworker.consumerArgs.InputChannel
	sig := sworker.consumerArgs.InputClosed.Channel()
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				sworker.logger.Info("input closed")
				return
			}
			if chunk.Data == nil {
				sworker.logger.Panicf("received unloaded chunk id=%s", chunk.ID)
			}
			sworker.saver.Write(chunk, sworker.decoder)
			sworker.consumerArgs.OnChunkConsumed(chunk)
		case <-sig:
			return
		}
	}
}
