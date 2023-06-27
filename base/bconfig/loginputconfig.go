package bconfig

import (
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
)

// LogInputConfig provides an interface for the configuration of LogInput(s)
//
// All the implementations should support YAML unmarshalling
type LogInputConfig interface {
	BaseConfig

	NewInput(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
		logBufferReceiver base.MultiSinkBufferReceiver, metricCreator promreg.MetricCreator,
		stopRequest channels.Awaitable) (base.LogInput, error)

	NewParser(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
		inputCounter *base.LogInputCounterSet) (base.LogParser, error)

	VerifyConfig(schema base.LogSchema) error
}

// LogInputConfigHolder holds LogInputConfig
type LogInputConfigHolder = ConfigHolder[LogInputConfig]

// LogInputConfigCreatorTable defines the table of constructors for LogInputConfig implementations
type LogInputConfigCreatorTable = ConfigCreatorTable[LogInputConfig]
