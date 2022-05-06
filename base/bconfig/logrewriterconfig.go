package bconfig

import (
	"github.com/relex/slog-agent/base"
)

// LogRewriterConfig provides an interface for the configuration of base.LogRewriter(s)
//
// All the implementations should support YAML unmarshalling
type LogRewriterConfig interface {
	BaseConfig

	NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter

	VerifyConfig(schema base.LogSchema, hasNext bool) error
}

type LogRewriterConfigHolder = ConfigHolder[LogRewriterConfig]
type LogRewriterConfigCreatorTable = ConfigCreatorTable[LogRewriterConfig]
