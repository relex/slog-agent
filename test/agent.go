package test

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/orchestrate/obykeyset"
	"github.com/relex/slog-agent/orchestrate/osingleton"
	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/util"
)

type agent struct {
	loader         *run.Loader
	inputAddresses []string
	shutdownFn     func()
	runRecovery    bool
}

func startAgent(loader *run.Loader, newChunkSaver base.ChunkConsumerConstructor, keysOverride []string, tagOverride string) *agent {
	if len(loader.Inputs) != 1 {
		logger.Warnf("only the first input is used for testing - there are %d", len(loader.Inputs))
	}
	switch ocfg := loader.Orchestration.OrchestratorConfig.(type) {
	case *obykeyset.Config:
		if keysOverride != nil {
			newMetricKeys := make([]string, 0, len(ocfg.Keys)+len(loader.MetricKeys))
			// move original orchestration Keys to config.MetricKeys
			for _, ok := range ocfg.Keys {
				if util.IndexOfString(keysOverride, ok) == -1 {
					newMetricKeys = append(newMetricKeys, ok)
				}
			}
			ocfg.Keys = keysOverride
			// remove dup keys from loader.MetricKeys
			for _, mk := range loader.MetricKeys {
				if util.IndexOfString(keysOverride, mk) == -1 {
					newMetricKeys = append(newMetricKeys, mk)
				}
			}
			loader.MetricKeys = newMetricKeys
			loader.PipelineArgs.MetricKeyLocators = loader.PipelineArgs.Schema.MustCreateFieldLocators(newMetricKeys)
		}
		if tagOverride != "" {
			ocfg.TagTemplate = tagOverride
		}
	case *osingleton.Config:
		if tagOverride != "" {
			ocfg.Tag = tagOverride
		}
	}

	// flush everything at the end if the output is not a real forwarder client
	if newChunkSaver != nil {
		loader.PipelineArgs.NewConsumerOverride = newChunkSaver
		loader.PipelineArgs.SendAllAtEnd = true
	}

	orchestrator := loader.LaunchOrchestrator(logger.Root())
	inputAddresses, shutdownInputFn := loader.LaunchInputs(orchestrator)

	return &agent{
		loader:         loader,
		inputAddresses: inputAddresses,
		shutdownFn: func() {
			shutdownInputFn()
			orchestrator.Shutdown()
		},
		runRecovery: newChunkSaver == nil,
	}
}

func (a *agent) Address() string {
	return a.inputAddresses[0]
}

func (a *agent) DumpMetrics() string {
	dump, err := a.loader.MetricFactory.DumpMetrics(false)
	if err != nil {
		logger.Panic(err)
	}
	return dump
}

func (a *agent) MetricFactory() *base.MetricFactory {
	return a.loader.MetricFactory
}

func (a *agent) StopAndWait() {
	a.shutdownFn()

	if !a.runRecovery {
		return
	}

	rlogger := logger.WithField("phase", "recovery")
	rlogger.Info("launch recovery orchestrator")

	a.loader.PipelineArgs.SendAllAtEnd = true

	// run recovery to send previously unsent chunks
	orchestrator := a.loader.LaunchOrchestrator(rlogger)
	orchestrator.Shutdown()
	rlogger.Info("recovery orchestrator done")
}
