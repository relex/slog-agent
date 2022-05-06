package bconfig

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

// LogTransformConfig provides an interface for the configuration of base.LogTransform(s)
//
// All the implementations should support YAML unmarshalling
type LogTransformConfig interface {
	BaseConfig

	NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform

	VerifyConfig(schema base.LogSchema) error
}

type LogTransformConfigHolder = ConfigHolder[LogTransformConfig]
type LogTransformConfigCreatorTable = ConfigCreatorTable[LogTransformConfig]
