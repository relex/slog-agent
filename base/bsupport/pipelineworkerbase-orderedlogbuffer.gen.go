// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package bsupport

import (
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// PipelineWorkerBaseForOrderedLogBuffer is the worker class / template for pipeline workers
// It contains an input channel, process function for each of input values, and "stop" signals which are triggered when input channel is closed
// The process function (PipelineProcFunc) is the only thing required from its composite parent
type PipelineWorkerBaseForOrderedLogBuffer struct {
	_baseLogger  logger.Logger
	_baseInput   <-chan base.OrderedLogBuffer
	_baseStopped *channels.SignalAwaitable
	_baseOnInput func(input base.OrderedLogBuffer, timeout <-chan time.Time)
	_baseOnTick  func(timeout <-chan time.Time)
	_baseOnStop  func(timeout <-chan time.Time)
}

// NewPipelineWorkerBaseForOrderedLogBuffer creates a new PipelineWorkerBaseForOrderedLogBuffer
func NewPipelineWorkerBaseForOrderedLogBuffer(logger logger.Logger, inputChannel <-chan base.OrderedLogBuffer) PipelineWorkerBaseForOrderedLogBuffer {
	return PipelineWorkerBaseForOrderedLogBuffer{
		_baseLogger:  logger,
		_baseInput:   inputChannel,
		_baseStopped: channels.NewSignalAwaitable(),
	}
}

// InitInternal initializes the internal function references called in processing loops
func (worker *PipelineWorkerBaseForOrderedLogBuffer) InitInternal(
	inputHandler func(input base.OrderedLogBuffer, timeout <-chan time.Time),
	tickHandler func(timeout <-chan time.Time),
	stopHandler func(timeout <-chan time.Time),
) {
	if worker._baseOnInput != nil {
		worker._baseLogger.Panic("re-initialization called")
	}
	worker._baseOnInput = inputHandler
	worker._baseOnTick = tickHandler
	worker._baseOnStop = stopHandler
}

// Launch starts the main loop of PipelineWorker in background
func (worker *PipelineWorkerBaseForOrderedLogBuffer) Launch() {
	go worker._baseRun()
}

// Logger returns the logger
func (worker *PipelineWorkerBaseForOrderedLogBuffer) Logger() logger.Logger {
	return worker._baseLogger
}

// Stopped returns an Awaitable which is signaled when stopped
func (worker *PipelineWorkerBaseForOrderedLogBuffer) Stopped() channels.Awaitable {
	return worker._baseStopped
}

func (worker *PipelineWorkerBaseForOrderedLogBuffer) _baseRun() {
	timeoutTimer := time.NewTimer(defs.IntermediateChannelTimeout)
	worker._baseProcessMain(timeoutTimer)

	if worker._baseOnTick != nil {
		util.ResetTimer(timeoutTimer, defs.IntermediateChannelTimeout)
		worker._baseOnTick(timeoutTimer.C)
	}
	if worker._baseOnStop != nil {
		util.ResetTimer(timeoutTimer, defs.IntermediateChannelTimeout)
		worker._baseOnStop(timeoutTimer.C)
	}
	timeoutTimer.Stop()
	worker._baseStopped.Signal()
}

// runMainLoop waits and processes incoming input until stop request, returns true for cleanup or false to abort
func (worker *PipelineWorkerBaseForOrderedLogBuffer) _baseProcessMain(timeoutTimer *time.Timer) {
	worker._baseLogger.Info("start main loop")
	ticker := time.NewTicker(defs.IntermediateFlushInterval)
SELECT_LOOP:
	for {
		select {
		case value, ok := <-worker._baseInput:
			if !ok {
				worker._baseLogger.Infof("end main loop on input channel close")
				break SELECT_LOOP
			}
			worker._baseOnInput(value, timeoutTimer.C)
		case <-ticker.C:
			if worker._baseOnTick != nil {
				worker._baseOnTick(timeoutTimer.C)
			}
			util.ResetTimer(timeoutTimer, defs.IntermediateChannelTimeout)
		}
	}
	ticker.Stop()
}
