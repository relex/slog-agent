package osingleton

import (
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

type singletonOrchestrator struct {
	logger       logger.Logger
	inputChannel chan []*base.LogRecord
	stopSignal   *channels.SignalAwaitable
}

type singletonOrchestratorChild struct {
	logger       logger.Logger
	inputChannel chan []*base.LogRecord
	sendTimeout  *time.Timer
}

// NewOrchestrator creates a singleton Orchestrator backed by one pipeline to aggregate and process all incoming logs
func NewOrchestrator(parentLogger logger.Logger, tag string, metricFactory *base.MetricFactory, launchWorkers base.PipelineWorkersLauncher) base.Orchestrator {
	o := &singletonOrchestrator{
		logger:       parentLogger.WithField(defs.LabelComponent, "SingletonOrchestrator"),
		inputChannel: make(chan []*base.LogRecord, defs.IntermediateBufferedChannelSize),
		stopSignal:   channels.NewSignalAwaitable(),
	}
	launchWorkers(o.logger, tag, "", o.inputChannel, metricFactory, o.stopSignal.Signal)
	return o
}

func (o *singletonOrchestrator) NewSink(clientAddress string, clientNumber base.ClientNumber) base.BufferReceiverSink {
	return &singletonOrchestratorChild{
		logger:       base.NewSinkLogger(o.logger, clientAddress, clientNumber),
		inputChannel: o.inputChannel,
		sendTimeout:  time.NewTimer(defs.IntermediateChannelTimeout),
	}
}

func (o *singletonOrchestrator) Shutdown() {
	close(o.inputChannel)
	o.stopSignal.WaitForever()
}

// Accept accepts input logs from LogInput, the buffer is only usable within the function
func (oc *singletonOrchestratorChild) Accept(buffer []*base.LogRecord) {
	reusableBuffer := bsupport.CopyLogBuffer(buffer)
	select {
	case oc.inputChannel <- reusableBuffer:
		// TODO: update metrics
		break
	case <-oc.sendTimeout.C:
		oc.logger.Errorf("BUG: timeout flushing: %d records. stack=%s", len(reusableBuffer), util.Stack())
		break
	}
}

// Tick renews internal timeout timer
func (oc *singletonOrchestratorChild) Tick() {
	util.ResetTimer(oc.sendTimeout, defs.IntermediateChannelTimeout)
}

// Close does nothing
func (oc *singletonOrchestratorChild) Close() {
	oc.logger.Info("close")
}
