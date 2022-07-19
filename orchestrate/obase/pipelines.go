package obase

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/util"
)

type outputWorkerSettings struct {
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

		outputSettingsSlice := util.MapSlice(args.OutputBufferPairs, func(pair bconfig.OutputBufferConfig) outputWorkerSettings {
			settings := outputWorkerSettings{
				bufferer: pair.BufferConfig.Value.NewBufferer(parentLogger, bufferID, pair.OutputConfig.Value.MatchChunkID,
					metricCreator, args.SendAllAtEnd),
			}

			// bufferer in the middle of pipeline has to be started first and shut down last for persistance of pending outputs
			settings.bufferer.Start()

			// then start output forwarder which is at the end of pipeline.
			// if there are queued logs from bufferer, the consumer would immediately start sending them.
			consumerLogger := parentLogger.WithField("output", pair.Name)
			if args.NewConsumerOverride != nil {
				consumerLogger.Info("launch override consumer")
				settings.consumer = args.NewConsumerOverride(consumerLogger, settings.bufferer.RegisterNewConsumer())
				settings.consumer.Start()
			} else {
				consumerLogger.Info("launch consumer")
				settings.consumer = pair.OutputConfig.Value.NewForwarder(
					consumerLogger,
					settings.bufferer.RegisterNewConsumer(),
					metricCreator.AddOrGetPrefix("output_", []string{"output"}, []string{pair.Name}),
				)
				settings.consumer.Start()
			}

			settings.serializer = pair.OutputConfig.Value.NewSerializer(parentLogger, args.Schema)
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
			util.MapSlice(outputSettingsSlice, func(outputSettings outputWorkerSettings) bsupport.OutputInterface {
				return bsupport.OutputInterface{
					LogSerializer: outputSettings.serializer,
					LogChunkMaker: outputSettings.chunkMaker,
					AcceptChunk:   outputSettings.bufferer.Accept,
				}
			}),
		)
		procWorker.Stopped().Next(func() {
			util.EachInSlice(outputSettingsSlice, func(_ int, settings outputWorkerSettings) {
				settings.bufferer.Destroy()
			})
			onStopped()
		})

		// only start the processing worker to handle incoming logs after all the rest have finished initialization
		// and their workers being started in background
		procWorker.Start()
	}
}
