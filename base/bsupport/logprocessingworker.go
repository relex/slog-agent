package bsupport

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// LogProcessingWorker is a worker for transformation, serialization and chunk making
type LogProcessingWorker struct {
	PipelineWorkerBase[[]*base.LogRecord]
	deallocator   *base.LogAllocator
	procCounter   *base.LogProcessCounter
	transformList []base.LogTransformFunc
	outputList    []OutputInterface
	lastChunkTime time.Time
}

type OutputInterface struct {
	base.LogSerializer
	base.LogChunkMaker
	AcceptChunk base.LogChunkAccepter
}

// NewLogProcessingWorker creates LogProcessingWorker
func NewLogProcessingWorker(parentLogger logger.Logger,
	input <-chan []*base.LogRecord, deallocator *base.LogAllocator, procCounter *base.LogProcessCounter,
	transforms []base.LogTransformFunc, components []OutputInterface) *LogProcessingWorker {
	worker := &LogProcessingWorker{
		PipelineWorkerBase: NewPipelineWorkerBase(
			parentLogger.WithField(defs.LabelComponent, "LogProcessingWorker"),
			input,
		),
		deallocator:   deallocator,
		procCounter:   procCounter,
		transformList: transforms,
		outputList:    components,
		lastChunkTime: time.Now(),
	}
	worker.InitInternal(worker.onInput, worker.onTick, worker.onStop)
	return worker
}

func (worker *LogProcessingWorker) onInput(buffer []*base.LogRecord, timeout <-chan time.Time) {
	if len(buffer) == 0 {
		return
	}
	for _, record := range buffer {
		icounter := worker.procCounter.SelectInputCounter(record)
		if RunTransforms(record, worker.transformList) == base.DROP {
			icounter.CountRecordDrop(record)
			worker.deallocator.Release(record)
			continue
		}
		icounter.CountRecordPass(record)

		for _, output := range worker.outputList {
			stream := output.SerializeRecord(record)
			worker.procCounter.CountStream(stream)
			maybeChunk := output.WriteStream(stream)
			if maybeChunk != nil {
				worker.procCounter.CountChunk(maybeChunk)
				output.AcceptChunk(*maybeChunk, timeout)
			}
		}
	}
}

func (worker *LogProcessingWorker) onTick(timeout <-chan time.Time) {
	// send buffered streams as a chunk if X seconds have passed
	if time.Since(worker.lastChunkTime) < defs.IntermediateFlushInterval {
		return
	}
	worker.flushChunk(timeout)
	worker.procCounter.UpdateMetrics()
}

func (worker *LogProcessingWorker) onStop(timeout <-chan time.Time) {
	worker.flushChunk(timeout)
	worker.procCounter.UpdateMetrics()
}

func (worker *LogProcessingWorker) flushChunk(timeout <-chan time.Time) {
	for _, output := range worker.outputList {
		maybeChunk := output.FlushBuffer()
		if maybeChunk != nil {
			worker.procCounter.CountChunk(maybeChunk)
			output.AcceptChunk(*maybeChunk, timeout)
		}
	}
}
