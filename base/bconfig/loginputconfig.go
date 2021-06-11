package bconfig

import (
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

// LogInputConfig provides an interface for the configuration of LogInput(s)
// All the implementations should support YAML unmarshalling
type LogInputConfig interface {
	GetType() string

	NewInput(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
		logBufferReceiver base.MultiChannelBufferReceiver, metricFactory *base.MetricFactory,
		stopRequest channels.Awaitable) (base.LogInput, error)

	NewParser(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
		inputCounter *base.LogInputCounter) (base.LogParser, error)

	VerifyConfig(schema base.LogSchema) error
}
