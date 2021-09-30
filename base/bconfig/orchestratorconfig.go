package bconfig

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
)

// OrchestratorConfig configures Orchestrator
type OrchestratorConfig interface {

	// GetType returns the type name
	GetType() string

	// LaunchOrchestrator creates and launches a new Orchestrator
	LaunchOrchestrator(parentLogger logger.Logger, args PipelineArgs, metricCreator promreg.MetricCreator) base.Orchestrator

	// VerifyConfig checks configuration and returns (key fields, error)
	VerifyConfig(schema base.LogSchema) ([]string, error)
}

// PipelineArgs defines the common arguments to construct pipeline(s)
type PipelineArgs struct {
	Schema              base.LogSchema
	Deallocator         *base.LogAllocator
	MetricKeyLocators   []base.LogFieldLocator
	TransformConfigs    []LogTransformConfigHolder    // Verified config list of transforms
	BufferConfig        ChunkBufferConfig             // Verified buffer config
	OutputConfig        LogOutputConfig               // Verified output config
	NewConsumerOverride base.ChunkConsumerConstructor // nil or override ChunkConsumer (ex: forwarder) for test
	SendAllAtEnd        bool                          // send all chunks to ChunkConsumer until consumed for test
}
