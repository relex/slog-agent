package obase

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
)

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

		// bufferer in the middle of pipeline has to be started first and shut down last for persistance of pending outputs
		outputBufferer := args.BufferConfig.NewBufferer(parentLogger, bufferID, args.OutputConfig.MatchChunkID,
			metricCreator, args.SendAllAtEnd)
		outputBufferer.Start()

		// then start output forwarder which is at the end of pipeline.
		// if there are queued logs from bufferer, the consumer would immediately start sending them.
		if args.NewConsumerOverride != nil {
			parentLogger.Info("launch override consumer")
			outputConsumer := args.NewConsumerOverride(parentLogger, outputBufferer.RegisterNewConsumer())
			outputConsumer.Start()
		} else {
			parentLogger.Info("launch consumer")
			outputConsumer := args.OutputConfig.NewForwarder(parentLogger, outputBufferer.RegisterNewConsumer(), metricCreator)
			outputConsumer.Start()
		}

		// then prepare processing worker which is at the head of pipeline
		procTracker := base.NewLogProcessCounter(metricCreator, args.Schema, args.MetricKeyLocators)
		procWorker := bsupport.NewLogProcessingWorker(
			parentLogger,
			input,
			args.Deallocator,
			procTracker,
			bsupport.NewTransformsFromConfig(args.TransformConfigs, args.Schema, parentLogger, procTracker),
			args.OutputConfig.NewSerializer(parentLogger, args.Schema, args.Deallocator),
			args.OutputConfig.NewChunkMaker(parentLogger, outputTag),
			outputBufferer.Accept,
		)
		procWorker.Stopped().Next(func() {
			outputBufferer.Destroy()
			onStopped()
		})

		// only start the processing worker to handle incoming logs after all the rest have finished initialization
		// and their workers being started in background
		procWorker.Start()
	}
}
