package bconfig

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
)

// OrchestratorConfig configures Orchestrator
type OrchestratorConfig interface {
	BaseConfig

	// StartOrchestrator creates and launches a new Orchestrator
	StartOrchestrator(parentLogger logger.Logger, args PipelineArgs, metricCreator promreg.MetricCreator) base.Orchestrator

	// VerifyConfig checks configuration and returns (key fields, error)
	VerifyConfig(schema base.LogSchema) ([]string, error)
}

// OrchestratorConfigHolder holds OrchestratorConfig
type OrchestratorConfigHolder = ConfigHolder[OrchestratorConfig]

// OrchestratorConfigCreatorTable defines the table of constructors for OrchestratorConfig implementations
type OrchestratorConfigCreatorTable = ConfigCreatorTable[OrchestratorConfig]

// PipelineArgs defines the common arguments to construct pipeline(s)
type PipelineArgs struct {
	Schema              base.LogSchema
	Deallocator         *base.LogAllocator
	MetricKeyLocators   []base.LogFieldLocator
	TransformConfigs    []LogTransformConfigHolder // Verified config list of transforms
	OutputBufferPairs   []OutputBufferConfig
	NewConsumerOverride base.ChunkConsumerConstructor // nil or override ChunkConsumer (ex: forwarder) for test
	SendAllAtEnd        bool                          // send all chunks to ChunkConsumer until consumed for test
}
