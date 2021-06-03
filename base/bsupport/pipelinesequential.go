package bsupport

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// NewSequentialPipelineLauncher creates a pipeline of transformer, serializer, chunk maker and chunk callback
func NewSequentialPipelineLauncher(args bconfig.PipelineArgs) base.PipelineWorkersLauncher {

	return func(parentLogger logger.Logger, tag string, pipelineID string, input <-chan []*base.LogRecord,
		metricFactory *base.MetricFactory, onStopped func()) {

		outputBufferer := args.BufferConfig.NewBufferer(parentLogger, pipelineID, args.OutputConfig.MatchChunkID,
			metricFactory, args.SendAllAtEnd)
		outputBufferer.Launch()

		if args.NewConsumerOverride != nil {
			parentLogger.Info("launch override consumer")
			outputConsumer := args.NewConsumerOverride(parentLogger, outputBufferer.RegisterNewConsumer())
			outputConsumer.Launch()
		} else {
			parentLogger.Info("launch consumer")
			outputConsumer := args.OutputConfig.NewForwarder(parentLogger, outputBufferer.RegisterNewConsumer(), metricFactory)
			outputConsumer.Launch()
		}

		procTracker := base.NewLogProcessCounter(metricFactory, args.Schema, args.MetricKeyLocators)

		procWorker := NewLogProcessingWorker(
			parentLogger,
			input,
			args.Deallocator,
			procTracker,
			NewTransformsFromConfig(args.TransformConfigs, args.Schema, parentLogger, procTracker),
			args.OutputConfig.NewSerializer(parentLogger, args.Schema, args.Deallocator),
			args.OutputConfig.NewChunkMaker(parentLogger, tag),
			outputBufferer.Accept,
		)
		procWorker.Launch()
		procWorker.Stopped().Next(func() {
			outputBufferer.Destroy()
			onStopped()
		})
	}
}
