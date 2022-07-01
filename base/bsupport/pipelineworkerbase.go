package bsupport

import (
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// PipelineWorkerBase is the base worker class for pipeline workers
//
// It contains an input channel, process function for each of input values, and "stop" signals which are triggered when input channel is closed
//
// The process function (PipelineProcFunc) is the only thing required from its composite parent
type PipelineWorkerBase[T any] struct {
	_baseLogger  logger.Logger
	_baseInput   <-chan T
	_baseStopped *channels.SignalAwaitable
	_baseOnInput func(input T, timeout <-chan time.Time)
	_baseOnTick  func(timeout <-chan time.Time)
	_baseOnStop  func(timeout <-chan time.Time)
}

// NewPipelineWorkerBase creates a new PipelineWorkerBase for specified data type in channel
func NewPipelineWorkerBase[T any](logger logger.Logger, inputChannel <-chan T) PipelineWorkerBase[T] {
	return PipelineWorkerBase[T]{
		_baseLogger:  logger,
		_baseInput:   inputChannel,
		_baseStopped: channels.NewSignalAwaitable(),
	}
}

// InitInternal initializes the internal function references called in processing loops
func (worker *PipelineWorkerBase[T]) InitInternal(
	inputHandler func(input T, timeout <-chan time.Time),
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
func (worker *PipelineWorkerBase[T]) Launch() {
	go worker._baseRun()
}

// Logger returns the logger
func (worker *PipelineWorkerBase[T]) Logger() logger.Logger {
	return worker._baseLogger
}

// Stopped returns an Awaitable which is signaled when stopped
func (worker *PipelineWorkerBase[T]) Stopped() channels.Awaitable {
	return worker._baseStopped
}

func (worker *PipelineWorkerBase[T]) _baseRun() {
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
func (worker *PipelineWorkerBase[T]) _baseProcessMain(timeoutTimer *time.Timer) {
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
