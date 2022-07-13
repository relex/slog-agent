package obase

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/util"
)

type pipelineWorkerSettings struct {
	bufferer   base.ChunkBufferer
	serializer base.LogSerializer
	chunkMaker base.LogChunkMaker
	consumer   base.ChunkConsumer
}

// PipelineStarter represents a function to launch workers for a top-level pipeline under Orchestrator
//
// bufferID must be unique inside the parent orchestrator
//
// Launched workers should start shutting down as soon as the input channel is closed and call onStopped at the end
type PipelineStarter func(parentLogger logger.Logger, metricCreator promreg.MetricCreator,
	input <-chan []*base.LogRecord, bufferID string, outputTag string, onStopped func())

// PrepareSequentialPipeline makes a starter for pipelines including transformer, serializer and output forwarder
func PrepareSequentialPipeline(args bconfig.PipelineArgs) PipelineStarter {

	return func(parentLogger logger.Logger, metricCreator promreg.MetricCreator,
		input <-chan []*base.LogRecord, bufferID string, outputTag string, onStopped func()) {

		outputSettingsSlice := util.MapSlice(args.OutputBufferPairs, func(pair bconfig.OutputBufferConfig) pipelineWorkerSettings {
			settings := pipelineWorkerSettings{
				bufferer: pair.BufferConfig.Value.NewBufferer(parentLogger, bufferID, pair.OutputConfig.Value.MatchChunkID,
					metricCreator, args.SendAllAtEnd),
			}

			// bufferer in the middle of pipeline has to be started first and shut down last for persistance of pending outputs
			settings.bufferer.Start()

			// then start output forwarder which is at the end of pipeline.
			// if there are queued logs from bufferer, the consumer would immediately start sending them.
			if args.NewConsumerOverride != nil {
				parentLogger.Info("launch override consumer")
				settings.consumer = args.NewConsumerOverride(parentLogger, settings.bufferer.RegisterNewConsumer())
				settings.consumer.Start()
			} else {
				parentLogger.Info("launch consumer")
				settings.consumer = pair.OutputConfig.Value.NewForwarder(parentLogger, settings.bufferer.RegisterNewConsumer(), metricCreator)
				settings.consumer.Start()
			}

			settings.serializer = pair.OutputConfig.Value.NewSerializer(parentLogger, args.Schema, args.Deallocator)
			settings.chunkMaker = pair.OutputConfig.Value.NewChunkMaker(parentLogger, outputTag)

			return settings
		})

		// then prepare processing worker which is at the head of pipeline
		procTracker := base.NewLogProcessCounter(metricCreator, args.Schema, args.MetricKeyLocators)
		procWorker := bsupport.NewLogProcessingWorker(
			parentLogger,
			input,
			args.Deallocator,
			procTracker,
			bsupport.NewTransformsFromConfig(args.TransformConfigs, args.Schema, parentLogger, procTracker),
			util.MapSlice(outputSettingsSlice, func(outputSettings pipelineWorkerSettings) bsupport.ProcessingWorkerOutputComponentSet {
				return bsupport.ProcessingWorkerOutputComponentSet{
					Serializer:  outputSettings.serializer,
					ChunkMaker:  outputSettings.chunkMaker,
					AcceptChunk: outputSettings.bufferer.Accept,
				}
			}),
		)
		procWorker.Stopped().Next(func() {
			util.EachInSlice(outputSettingsSlice, func(_ int, settings pipelineWorkerSettings) {
				settings.bufferer.Destroy()
			})
			onStopped()
		})

		// only start the processing worker to handle incoming logs after all the rest have finished initialization
		// and their workers being started in background
		procWorker.Start()
	}
}
