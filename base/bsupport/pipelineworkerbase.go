package bsupport

import (
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/defs"
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
	_baseOnInput func(input T)
	_baseOnTick  func()
	_baseOnStop  func()
}

// NewPipelineWorkerBase creates a new PipelineWorkerBase for specified data type in channel
func NewPipelineWorkerBase[T any](baseLogger logger.Logger, inputChannel <-chan T) PipelineWorkerBase[T] {
	return PipelineWorkerBase[T]{
		_baseLogger:  baseLogger,
		_baseInput:   inputChannel,
		_baseStopped: channels.NewSignalAwaitable(),
	}
}

// InitInternal initializes the internal function references called in processing loops
func (worker *PipelineWorkerBase[T]) InitInternal(
	inputHandler func(input T),
	tickHandler func(),
	stopHandler func(),
) {
	if worker._baseOnInput != nil {
		worker._baseLogger.Panic("re-initialization called")
	}
	worker._baseOnInput = inputHandler
	worker._baseOnTick = tickHandler
	worker._baseOnStop = stopHandler
}

// Start starts the main loop of PipelineWorker in background
func (worker *PipelineWorkerBase[T]) Start() {
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
	worker._baseProcessMain()

	if worker._baseOnTick != nil {
		worker._baseOnTick()
	}
	if worker._baseOnStop != nil {
		worker._baseOnStop()
	}
	worker._baseStopped.Signal()
}

// runMainLoop waits and processes incoming input until stop request, returns true for cleanup or false to abort
func (worker *PipelineWorkerBase[T]) _baseProcessMain() {
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
			worker._baseOnInput(value)
		case <-ticker.C:
			if worker._baseOnTick != nil {
				worker._baseOnTick()
			}
		}
	}
	ticker.Stop()
}
