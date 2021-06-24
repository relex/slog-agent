package run

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

// ReloadableOrchestrator is a wrapper of ordinary Orchestrator(s) to support re-creation of the Orchestrator on config reload
//
// The type is to be paired with Reloader, which provides the function to reload configuration file and create real Orchestrator(s)
type ReloadableOrchestrator struct {
	downstream      base.Orchestrator        // the real orchestrator
	renewDownstream func() base.Orchestrator // function to perform reloading and launch a new downstream orchestrator
}

func NewReloadableOrchestrator(downstream base.Orchestrator, renewDownstream func() base.Orchestrator) *ReloadableOrchestrator {
	// DO NOT use renewDownstream for initial downstream creation, as config reloading is very different from first-time loading

	// handle signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			logger.Info("Reloading config...")
		}
	}()

	// TODO: renew downstream on signal

	return &ReloadableOrchestrator{
		downstream:      downstream,
		renewDownstream: renewDownstream,
	}
}

func (orc *ReloadableOrchestrator) NewChannel(id string) base.BufferReceiverChannel {
	return &ReloadableChannel{
		downstream: orc.downstream.NewChannel(id),
	}
}

func (orc *ReloadableOrchestrator) Shutdown() {
	orc.downstream.Shutdown()
}

type ReloadableChannel struct {
	downstream base.BufferReceiverChannel
}

func (channel *ReloadableChannel) Accept(buffer []*base.LogRecord) {
	channel.downstream.Accept(buffer)
}

func (channel *ReloadableChannel) Tick() {
	channel.downstream.Tick()
}

func (channel *ReloadableChannel) Close() {
	channel.downstream.Close()
}
