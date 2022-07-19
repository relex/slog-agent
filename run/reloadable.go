package run

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/puzpuzpuz/xsync"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// sinksByClientNumber is a fix-sized array to hold downstream sinks by client number as array index
type sinksByClientNumber [base.MaxClientNumber]base.BufferReceiverSink

// addrsByClientNumber is a fix-sized array to hold client addresses by client number as array index
type addrsByClientNumber [base.MaxClientNumber]string

// InitiateReloadingFunc initiates the reloading process and returns error if it cannot be continued (e.g. error in
// new config)
//
// No reload would happen and there should be no side effect if the returned CompleteReloadingFunc is not called
type InitiateReloadingFunc func() (CompleteReloadingFunc, error)

// CompleteReloadingFunc completes the reloading process and launches the new downstream Orchestrator
//
// The resulting Orchestrator must be used to replace the existing one after this function is called
type CompleteReloadingFunc func() base.Orchestrator

// ReloadableOrchestrator supports config reload by re-creating the downstream Orchestrator
//
// The type is to be paired with Reloader, which provides the function to reload configuration file and create real Orchestrator(s)
type ReloadableOrchestrator struct {
	logger          logger.Logger
	downstream      base.Orchestrator     // the real orchestrator
	initiateReload  InitiateReloadingFunc // function to start reloading and launch a new downstream orchestrator
	downstreamSinks sinksByClientNumber
	downstreamAddrs addrsByClientNumber
	downstreamMutex *xsync.RBMutex // read-lock for using/adding downstream sinks, write-lock for renewing the downstream Orchestrator
}

// NewReloadableOrchestrator creates a reloadable orchestrator wrapping the given downstream orchestrator
func NewReloadableOrchestrator(downstream base.Orchestrator, initiateReload InitiateReloadingFunc) *ReloadableOrchestrator {
	// DO NOT use renewDownstream for initial downstream creation, as config reloading is very different from first-time loading

	rorc := &ReloadableOrchestrator{
		logger:          logger.WithField(defs.LabelComponent, "ReloadableOrchestrator"),
		downstream:      downstream,
		initiateReload:  initiateReload,
		downstreamSinks: sinksByClientNumber{},
		downstreamAddrs: addrsByClientNumber{},
		downstreamMutex: &xsync.RBMutex{},
	}

	// listen to signal: unbuffered since we want to discard any new SIGHUPs while reloading
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			// wait for SIGHUP
			<-c
			// handle SIGHUP
			rorc.reload()
		}
	}()

	return rorc
}

// NewSink creates a new reloadable sink for an input source (e.g. incoming TCP connection)
func (orc *ReloadableOrchestrator) NewSink(clientAddress string, clientNumber base.ClientNumber) base.BufferReceiverSink {
	newDownstream := orc.downstream.NewSink(clientAddress, clientNumber)

	lockT := orc.downstreamMutex.RLock() // only read-lock since we assume clientNumber is unique and nobody else is accessing it
	defer orc.downstreamMutex.RUnlock(lockT)

	if orc.downstreamSinks[clientNumber] != nil {
		orc.logger.WithFields(logger.Fields{
			"newClient":    clientAddress,
			"oldClient":    orc.downstreamAddrs[clientNumber],
			"clientNumber": clientNumber,
		}).Error("created new sink while old sink is still in place")
	}
	orc.downstreamSinks[clientNumber] = newDownstream
	orc.downstreamAddrs[clientNumber] = clientAddress

	return &ReloadableSink{
		downstreamPtr:   &orc.downstreamSinks[clientNumber],
		downstreamMutex: orc.downstreamMutex,
	}
}

// Shutdown shuts down both of the reloadable orchestrator and the current downstream orchestrator
func (orc *ReloadableOrchestrator) Shutdown() {
	orc.downstream.Shutdown()
}

func (orc *ReloadableOrchestrator) reload() {
	orc.logger.Info("reloading config...")
	completeRenewal, renewalErr := orc.initiateReload()
	if renewalErr != nil {
		orc.logger.Error("failed to reload due to: ", renewalErr)
		reloadFailureCounter.Inc()
		return
	}

	// wait and then block all ReloadableSink(s)
	orc.downstreamMutex.Lock()
	defer orc.downstreamMutex.Unlock()

	// close sinks created with old configuration and shut down
	for _, sink := range orc.downstreamSinks {
		if sink == nil {
			continue
		}
		sink.Close()
		// keep closed sinks in place so we know which ones to re-create below
	}
	orc.downstream.Shutdown()

	// recreate downstream Orchestrator and all sinks closed above
	orc.downstream = completeRenewal()
	for i, sink := range orc.downstreamSinks {
		if sink == nil {
			continue
		}
		clientAddress := orc.downstreamAddrs[i]
		clientNumber := base.ClientNumber(i)
		orc.downstreamSinks[i] = orc.downstream.NewSink(clientAddress, clientNumber)
	}
	orc.logger.Info("reloaded config")
	reloadSuccessCounter.Inc()
}

// ReloadableSink wraps Orchestrator's BufferReceiverSink to support reloading
//
// Like BufferReceiverSink, a ReloadableSink runs in individual input goroutines (e.g. TCP connection handler)
type ReloadableSink struct {
	downstreamPtr   *base.BufferReceiverSink
	downstreamMutex *xsync.RBMutex
}

// Accept passes buffered logs to the bound downstream sink
func (sink *ReloadableSink) Accept(buffer []*base.LogRecord) {
	lockT := sink.downstreamMutex.RLock()
	defer sink.downstreamMutex.RUnlock(lockT)

	(*sink.downstreamPtr).Accept(buffer)
}

// Tick calls Tick on the bound downstream sink
func (sink *ReloadableSink) Tick() {
	lockT := sink.downstreamMutex.RLock()
	defer sink.downstreamMutex.RUnlock(lockT)

	(*sink.downstreamPtr).Tick()
}

// Close closes the bound downstream sink and the reloadable sink itself
func (sink *ReloadableSink) Close() {
	lockT := sink.downstreamMutex.RLock()
	defer sink.downstreamMutex.RUnlock(lockT)

	(*sink.downstreamPtr).Close()
	*sink.downstreamPtr = nil
}
