package bsupport

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// ParallelDistributorLauncher defines a function to launch distributor for parallel pipelines
type ParallelDistributorLauncher func(parentLogger logger.Logger, input <-chan []*base.LogRecord, tag string,
	metricFactory *base.MetricFactory, launchChildWorkers base.OrderedPipelineWorkersLauncher, onStopped func())

// NewParallelPipelineLauncher creates a parallel pipeline for processing, buffering and consuming logs
//
// Each pipeline has its own bufferer and consumer, while actual processing is distributed among child pipelines
// Each child pipeline has its own transformer, serializer and chunk maker
func NewParallelPipelineLauncher(args bconfig.PipelineArgs, launchDistributor ParallelDistributorLauncher) base.PipelineWorkersLauncher {

	return func(parentLogger logger.Logger, tag string, pipelineID string, input <-chan []*base.LogRecord,
		metricFactory *base.MetricFactory, onStopped func()) {

		outputBufferer := args.BufferConfig.NewBufferer(parentLogger, pipelineID, args.OutputConfig.MatchChunkID,
			metricFactory, args.SendAllAtEnd)
		outputBufferer.Launch()

		if args.NewConsumerOverride != nil {
			outputConsumer := args.NewConsumerOverride(parentLogger, outputBufferer.RegisterNewConsumer())
			outputConsumer.Launch()
		} else {
			outputConsumer := args.OutputConfig.NewForwarder(parentLogger, outputBufferer.RegisterNewConsumer(), metricFactory)
			outputConsumer.Launch()
		}

		// all parallel workers within distributor shares same buffer and same consumer
		launchChildWorkers := newOrderedPipelineLauncher(args, outputBufferer.Accept)
		launchDistributor(parentLogger, input, tag, metricFactory, launchChildWorkers, func() {
			outputBufferer.Destroy()
			onStopped()
		})
	}
}

// newOrderedPipelineLauncher creates a pipeline for processing logs, with locks to ensure the order of log batches across multiple pipelines
//
// This is used under a parallel pipeline and not needed unless the order of log batches need to be restored after orchestration
func newOrderedPipelineLauncher(args bconfig.PipelineArgs, chunkAccepter base.LogChunkAccepter) base.OrderedPipelineWorkersLauncher {

	return func(parentLogger logger.Logger, tag string, pipelineNum int, input <-chan base.OrderedLogBuffer, // pipelineNum as ID is unused here
		metricFactory *base.MetricFactory, onStopped func()) {

		procTracker := base.NewLogProcessCounter(metricFactory, args.Schema, args.MetricKeyLocators)

		procWorker := NewOrderedLogProcessingWorker(
			parentLogger,
			input,
			args.Deallocator,
			procTracker,
			NewTransformsFromConfig(args.TransformConfigs, args.Schema, parentLogger, procTracker),
			args.OutputConfig.NewSerializer(parentLogger, args.Schema, args.Deallocator),
			args.OutputConfig.NewChunkMaker(parentLogger, tag),
			chunkAccepter,
		)
		procWorker.Launch()
		procWorker.Stopped().Next(onStopped)
	}
}
