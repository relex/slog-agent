package test

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/orchestrate/obykeyset"
	"github.com/relex/slog-agent/orchestrate/osingleton"
	"github.com/relex/slog-agent/run"
	"golang.org/x/exp/slices"
)

type agent struct {
	loader         *run.Loader
	inputAddresses []string
	shutdownFn     func()
	runRecovery    bool
}

// startAgent starts the integration-testing log agent which is different from production mode in several ways:
//
// 1. output chunks may be intercepted before real/network forwarder. If intercepted, the bufferer flushes everything before shutdown instead of saving them for recovery.
//
// 2. orchestration keys and tags from config may be overridden
func startAgent(loader *run.Loader, newChunkSaver base.ChunkConsumerConstructor, keysOverride []string, tagOverride string) *agent {
	if len(loader.Inputs) != 1 {
		logger.Warnf("only the first input is used for testing - there are %d", len(loader.Inputs))
	}
	switch orcConf := loader.Orchestration.Value.(type) {
	case *obykeyset.Config:
		if keysOverride != nil {
			newMetricKeys := make([]string, 0, len(orcConf.Keys)+len(loader.MetricKeys))
			// move original orchestration Keys to config.MetricKeys
			for _, ok := range orcConf.Keys {
				if slices.Index(keysOverride, ok) == -1 {
					newMetricKeys = append(newMetricKeys, ok)
				}
			}
			orcConf.Keys = keysOverride
			// remove dup keys from loader.MetricKeys
			for _, mk := range loader.MetricKeys {
				if slices.Index(keysOverride, mk) == -1 {
					newMetricKeys = append(newMetricKeys, mk)
				}
			}
			loader.MetricKeys = newMetricKeys
			loader.PipelineArgs.MetricKeyLocators = loader.PipelineArgs.Schema.MustCreateFieldLocators(newMetricKeys)
		}
		if tagOverride != "" {
			orcConf.TagTemplate = tagOverride
		}
	case *osingleton.Config:
		if tagOverride != "" {
			orcConf.Tag = tagOverride
		}
	default:
		logger.Panic("unsupported orchestrator type: ", orcConf)
	}

	// flush everything at the end if the output is not a real forwarder client
	if newChunkSaver != nil {
		loader.PipelineArgs.NewConsumerOverride = newChunkSaver
		loader.PipelineArgs.SendAllAtEnd = true
	}

	orchestrator := loader.StartOrchestrator(logger.Root())
	inputAddresses, shutdownInputFn := loader.LaunchInputs(orchestrator)

	return &agent{
		loader:         loader,
		inputAddresses: inputAddresses,
		shutdownFn: func() {
			shutdownInputFn()
			time.Sleep(1 * time.Second) // give orchestrator time to process flushed logs at the end of tcp listener
			orchestrator.Shutdown()
		},
		runRecovery: newChunkSaver == nil,
	}
}

func (a *agent) Address() string {
	return a.inputAddresses[0]
}

func (a *agent) GetMetricQuerier() promreg.MetricQuerier {
	return a.loader.GetMetricQuerier()
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
	orchestrator := a.loader.RelaunchOrchestrator(rlogger)
	orchestrator.Shutdown()
	rlogger.Info("recovery orchestrator done")
}
