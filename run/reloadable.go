package run

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// sinksByClientNumber is a fix-sized array to hold downstream sinks by client number as array index
type sinksByClientNumber [base.MaxClientNumber]base.BufferReceiverSink

// addrsByClientNumber is a fix-sized array to hold client addresses by client number as array index
type addrsByClientNumber [base.MaxClientNumber]string

// ReloadableOrchestrator supports config reload by re-creating the downstream Orchestrator
//
// The type is to be paired with Reloader, which provides the function to reload configuration file and create real Orchestrator(s)
type ReloadableOrchestrator struct {
	paused          int32
	downstream      base.Orchestrator        // the real orchestrator
	renewDownstream func() base.Orchestrator // function to perform reloading and launch a new downstream orchestrator
	downstreamSinks sinksByClientNumber
	downstreamAddrs addrsByClientNumber
	reloadLock      *sync.RWMutex
}

// NewReloadableOrchestrator creates a reloadable orchestrator wrapping the given downstream orchestrator
func NewReloadableOrchestrator(downstream base.Orchestrator, renewDownstream func() base.Orchestrator) *ReloadableOrchestrator {
	// DO NOT use renewDownstream for initial downstream creation, as config reloading is very different from first-time loading

	rorc := &ReloadableOrchestrator{
		paused:          0,
		downstream:      downstream,
		renewDownstream: renewDownstream,
		downstreamSinks: sinksByClientNumber{},
		downstreamAddrs: addrsByClientNumber{},
		reloadLock:      &sync.RWMutex{},
	}

	// listen to signal: unbuffered since we can ignore new SIGHUPs while reloading
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			// wait for SIGHUP
			<-c
			// handle SIGHUP
			logger.Info("Reloading config...")
			rorc.reload()
			logger.Info("Reloaded config")
		}
	}()

	return rorc
}

// NewSink creates a new reloadable sink for an input source
func (orc *ReloadableOrchestrator) NewSink(clientAddress string, clientNumber base.ClientNumber) base.BufferReceiverSink {
	orc.waitForReload()

	newSink := orc.downstream.NewSink(clientAddress, clientNumber)

	if orc.downstreamSinks[clientNumber] != nil {
		logger.WithFields(logger.Fields{
			defs.LabelComponent: "ReloadableOrchestrator",
			"newClient":         clientAddress,
			"oldClient":         orc.downstreamAddrs[clientNumber],
			"clientNumber":      clientNumber,
		}).Error("created new sink while old sink is still in place")
	}
	orc.downstreamSinks[clientNumber] = newSink
	orc.downstreamAddrs[clientNumber] = clientAddress

	return &ReloadableSink{
		parent:        orc,
		downstreamPtr: &orc.downstreamSinks[clientNumber],
	}
}

// Shutdown shuts down both of the reloadable orchestrator and the current downstream orchestrator
func (orc *ReloadableOrchestrator) Shutdown() {
	orc.downstream.Shutdown()
}

func (orc *ReloadableOrchestrator) reload() {
	orc.reloadLock.Lock()         // obtain write lock, blocking waitForReload()
	defer orc.reloadLock.Unlock() // release write lock at the end to wake up all waitForReload() callers

	atomic.StoreInt32(&orc.paused, 1)

	// Close sinks created with old configuration and shut down

	for _, sink := range orc.downstreamSinks {
		if sink == nil {
			continue
		}
		sink.Close()
		// Keep closed sinks so we know which ones to re-create below
	}

	orc.downstream.Shutdown()

	// Recreate downstream Orchestrator and all sinks closed above

	orc.downstream = orc.renewDownstream()

	for i, sink := range orc.downstreamSinks {
		if sink == nil {
			continue
		}
		clientAddress := orc.downstreamAddrs[i]
		clientNumber := base.ClientNumber(i)
		orc.downstreamSinks[i] = orc.downstream.NewSink(clientAddress, clientNumber)
	}
}

func (orc *ReloadableOrchestrator) waitForReload() {
	if atomic.LoadInt32(&orc.paused) == 0 {
		return
	}

	orc.reloadLock.RLock()   // wait for write-lock to be released in reload()
	orc.reloadLock.RUnlock() // release immediately TODO: use Cond or something proper
}

// ReloadableSink wraps Orchestrator's BufferReceiverSink to support reloading
type ReloadableSink struct {
	parent        *ReloadableOrchestrator
	downstreamPtr *base.BufferReceiverSink
}

// Accept passes buffered logs to the current downstream sink
func (sink *ReloadableSink) Accept(buffer []*base.LogRecord) {
	sink.parent.waitForReload()

	(*sink.downstreamPtr).Accept(buffer)
}

// Tick calls Tick on the current downstream sink
func (sink *ReloadableSink) Tick() {
	sink.parent.waitForReload()

	(*sink.downstreamPtr).Tick()
}

// Close closes the current downstream sink and the reloadable sink itself
func (sink *ReloadableSink) Close() {
	sink.parent.waitForReload()

	(*sink.downstreamPtr).Close()
	*sink.downstreamPtr = nil
}
