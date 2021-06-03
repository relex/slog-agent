package test

import (
	"fmt"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/orchestrate/obykeyset"
	"github.com/relex/slog-agent/orchestrate/osingleton"
	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/util"
)

type agent struct {
	config      *run.Config
	schema      base.LogSchema
	mfactory    *base.MetricFactory
	input       base.LogInput
	stopReq     *channels.SignalAwaitable
	runRecovery bool
}

func startAgent(config *run.Config, schema base.LogSchema, newChunkSaver base.ChunkConsumerConstructor, keysOverride []string, tagOverride string) (*agent, error) {
	if len(config.Inputs) != 1 {
		return nil, fmt.Errorf("only one input is supported - there are %d", len(config.Inputs))
	}
	switch ocfg := config.Orchestration.OrchestratorConfig.(type) {
	case *obykeyset.Config:
		if keysOverride != nil {
			newMetricKeys := make([]string, 0, len(ocfg.Keys)+len(config.MetricKeys))
			// move original orchestration Keys to config.MetricKeys
			for _, ok := range ocfg.Keys {
				if util.IndexOfString(keysOverride, ok) == -1 {
					newMetricKeys = append(newMetricKeys, ok)
				}
			}
			ocfg.Keys = keysOverride
			// remove dup keys from config.MetricKeys
			for _, mk := range config.MetricKeys {
				if util.IndexOfString(keysOverride, mk) == -1 {
					newMetricKeys = append(newMetricKeys, mk)
				}
			}
			config.MetricKeys = newMetricKeys
		}
		if tagOverride != "" {
			ocfg.TagTemplate = tagOverride
		}
	case *osingleton.Config:
		if tagOverride != "" {
			ocfg.Tag = tagOverride
		}
	}
	allocator := base.NewLogAllocator(schema)
	pipelineArgs := bconfig.PipelineArgs{
		Schema:              schema,
		Deallocator:         allocator,
		MetricKeyLocators:   schema.MustCreateFieldLocators(config.MetricKeys),
		TransformConfigs:    config.Transformations,
		BufferConfig:        config.Buffer,
		OutputConfig:        config.Output,
		NewConsumerOverride: newChunkSaver,
		SendAllAtEnd:        newChunkSaver != nil, // send everything is not a real forwarder client
	}
	mfactory := base.NewMetricFactory("testagent_", nil, nil)
	orchestrator := config.Orchestration.LaunchOrchestrator(logger.Root(), pipelineArgs, mfactory)

	stopRequest := channels.NewSignalAwaitable()
	input, ierr := config.Inputs[0].NewInput(logger.Root(), allocator, schema, orchestrator, mfactory, stopRequest)
	if ierr != nil {
		return nil, fmt.Errorf("input[0]: %w", ierr)
	}
	input.Launch()

	return &agent{
		config:      config,
		schema:      schema,
		mfactory:    mfactory,
		input:       input,
		stopReq:     stopRequest,
		runRecovery: newChunkSaver == nil,
	}, nil
}

func (a *agent) Address() string {
	return a.input.Address()
}

func (a *agent) StopAndWait() {
	a.stopReq.Signal()
	a.input.Stopped().WaitForever()

	if !a.runRecovery {
		return
	}

	rlogger := logger.WithField("phase", "recovery")
	rlogger.Info("launch recovery orchestrator")

	allocator := base.NewLogAllocator(a.schema)
	pipelineArgs := bconfig.PipelineArgs{
		Schema:              a.schema,
		Deallocator:         allocator,
		MetricKeyLocators:   a.schema.MustCreateFieldLocators(a.config.MetricKeys),
		TransformConfigs:    a.config.Transformations,
		BufferConfig:        a.config.Buffer,
		OutputConfig:        a.config.Output,
		NewConsumerOverride: nil,
		SendAllAtEnd:        true,
	}

	// run recovery to send previously unsent chunks
	orchestrator := a.config.Orchestration.LaunchOrchestrator(rlogger, pipelineArgs, a.mfactory)

	orchestrator.Destroy()
	rlogger.Info("recovery orchestrator done")
}

func (a *agent) DumpMetrics() string {
	dump, err := a.mfactory.DumpMetrics(false)
	if err != nil {
		logger.Panic(err)
	}
	return dump
}
