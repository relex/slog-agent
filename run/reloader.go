package run

import (
	"sync"
	"sync/atomic"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// Reloader overrides Loader to support configuration reloading
//
// It provides the underlying configuration reloading but not responsible to trigger or to use the reloaded setup
type Reloader struct {
	*Loader

	loadingLock sync.Mutex
}

func NewReloaderFromConfigFile(filepath string, metricPrefix string) (*Reloader, error) {
	// launch the orchestrator which manages pipelines
	loader, loaderErr := NewLoaderFromConfigFile(filepath, metricPrefix)
	if loaderErr != nil {
		return nil, loaderErr // return error if config file is invalid at startup (NOT when reloading)
	}

	return &Reloader{
		Loader: loader,
	}, nil
}

// LaunchOrchestrator launches a reloadable Orchestrator
func (reloader *Reloader) LaunchOrchestrator(ologger logger.Logger) base.Orchestrator {
	firstDownstreamOrchestrator := reloader.Loader.LaunchOrchestrator(ologger)
	var numReload int64 = 0

	return NewReloadableOrchestrator(firstDownstreamOrchestrator, func() base.Orchestrator {
		// assume multiple reloadings don't happen in very short period to trigger concurrency issues
		reloader.reloadConfigFile()
		return reloader.Loader.LaunchOrchestrator(ologger.WithField("numReload", atomic.AddInt64(&numReload, 1)))
	})
}

func (reloader *Reloader) reloadConfigFile() {
	reloader.loadingLock.Lock()
	defer reloader.loadingLock.Unlock()

	newLoader, newErr := NewLoaderFromConfigFile(reloader.filepath, reloader.MetricFactory.Prefix())
	if newErr != nil {
		logger.WithField(defs.LabelComponent, "Reloader").Error("failed to reload: ", newErr)
		return
	}

	newLoader.Schema = reloader.Loader.Schema
	newLoader.PipelineArgs.Deallocator = reloader.Loader.PipelineArgs.Deallocator
	newLoader.MetricFactory = reloader.Loader.MetricFactory
	reloader.Loader = newLoader
}
