package base

import (
	"github.com/relex/gotils/channels"
)

// PipelineWorker represents a background worker in a stage of the processing pipeline, e.g. a parser or transformer
type PipelineWorker interface {
	Start()
	Stopped() channels.Awaitable
}
