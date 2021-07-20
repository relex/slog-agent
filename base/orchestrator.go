package base

import (
	"github.com/relex/gotils/logger"
)

// Orchestrator takes log records and distribute them to internal pipelines
type Orchestrator interface {
	MultiSinkBufferReceiver

	// Shutdown performs cleanup; should be called after all inputs have been stopped
	Shutdown()
}

// PipelineWorkersLauncher represents a function to launch workers for a top-level pipeline under Orchestrator
//
// pipelineID is unique inside the parent orchestrator;
// Launched workers should start shutting down as soon as the input channel is closed and call onStopped at the end
type PipelineWorkersLauncher func(parentLogger logger.Logger, tag string, pipelineID string, input <-chan []*LogRecord,
	metricFactory *MetricFactory, onStopped func())

// OrderedPipelineWorkersLauncher represents a function to launch child pipeline workers under a top-level parallel pipeline
//
// pipelineNum is unique inside the parent pipeline;
// Launched workers should start shutting down as soon as the input channel is closed and call onStopped at the end
type OrderedPipelineWorkersLauncher func(parentLogger logger.Logger, tag string, pipelineNum int, input <-chan OrderedLogBuffer,
	metricFactory *MetricFactory, onStopped func())
