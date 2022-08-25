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
	name       string
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
		input <-chan []*base.LogRecord, bufferID string, outputTag string, onStopped func(),
	) {
		outputSettingsSlice := util.MapSlice(args.OutputBufferPairs, func(pair bconfig.OutputBufferConfig) outputWorkerSettings {
			outputLogger := parentLogger.WithField("output", pair.Name)

			bufferer := pair.BufferConfig.Value.NewBufferer(
				outputLogger,
				bufferID,
				pair.OutputConfig.Value.MatchChunkID,
				metricCreator.AddOrGetPrefix("buffer_", []string{"output"}, []string{pair.Name}),
				args.SendAllAtEnd)

			// bufferer in the middle of pipeline has to be started first and shut down last for persistence of pending outputs
			bufferer.Start()

			// then start output forwarder which is at the end of pipeline.
			// if there are queued logs from bufferer, the consumer would immediately start sending them.
			var consumer base.ChunkConsumer
			if args.NewConsumerOverride != nil {
				outputLogger.Info("launch override consumer")
				consumer = args.NewConsumerOverride(outputLogger, bufferer.RegisterNewConsumer())
			} else {
				outputLogger.Info("launch consumer")
				consumer = pair.OutputConfig.Value.NewForwarder(
					outputLogger,
					bufferer.RegisterNewConsumer(),
					metricCreator.AddOrGetPrefix("output_", []string{"output"}, []string{pair.Name}),
				)
			}
			consumer.Start()

			return outputWorkerSettings{
				name:       pair.Name,
				bufferer:   bufferer,
				serializer: pair.OutputConfig.Value.NewSerializer(outputLogger, args.Schema),
				chunkMaker: pair.OutputConfig.Value.NewChunkMaker(outputLogger, outputTag),
				consumer:   consumer,
			}
		})

		// then prepare processing worker which is at the head of pipeline
		// output names slice order should match the outputs order in bsupport.NewLogProcessingWorker
		procTracker := base.NewLogProcessCounter(metricCreator, args.Schema, args.MetricKeyLocators,
			util.MapSlice(outputSettingsSlice, func(outputSettings outputWorkerSettings) string { return outputSettings.name }))

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
					Name:          outputSettings.name,
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
