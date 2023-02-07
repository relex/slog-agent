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

// OutputInterface is a joint interface of output components
type OutputInterface struct {
	base.LogSerializer
	base.LogChunkMaker
	Name        string
	AcceptChunk base.LogChunkAccepter
}

// NewLogProcessingWorker creates LogProcessingWorker
func NewLogProcessingWorker(parentLogger logger.Logger,
	input <-chan []*base.LogRecord, deallocator *base.LogAllocator, procCounter *base.LogProcessCounter,
	transforms []base.LogTransformFunc, outputInterfaces []OutputInterface,
) *LogProcessingWorker {
	worker := &LogProcessingWorker{
		PipelineWorkerBase: NewPipelineWorkerBase(
			parentLogger.WithField(defs.LabelComponent, "LogProcessingWorker"),
			input,
		),
		deallocator:   deallocator,
		procCounter:   procCounter,
		transformList: transforms,
		outputList:    outputInterfaces,
		lastChunkTime: time.Now(),
	}
	worker.InitInternal(worker.onInput, worker.onTick, worker.onStop)
	return worker
}

func (worker *LogProcessingWorker) onInput(buffer []*base.LogRecord) {
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

		for i, output := range worker.outputList {
			// TODO:
			//if RunTransforms(record, output.transformList) == base.DROP {
			//	icounter.CountOutputFilter(i, record)
			//	worker.deallocator.Release(record)
			//	continue
			//}
			stream := output.SerializeRecord(record)
			// TODO: decide whether to release once at the end or release here after per-output transform is implemented
			// It will depend on whether records are duplicated for additional outputs, or the same record with all transforms run in place.
			worker.deallocator.Release(record)
			worker.procCounter.CountStream(i, stream)
			maybeChunk := output.WriteStream(stream)
			if maybeChunk != nil {
				worker.procCounter.CountChunk(i, maybeChunk)
				output.AcceptChunk(*maybeChunk)
			}
		}
	}
}

func (worker *LogProcessingWorker) onTick() {
	// send buffered streams as a chunk if X seconds have passed
	if time.Since(worker.lastChunkTime) < defs.IntermediateFlushInterval {
		return
	}
	worker.flushChunk()
	worker.procCounter.UpdateMetrics()
}

func (worker *LogProcessingWorker) onStop() {
	worker.flushChunk()
	worker.procCounter.UpdateMetrics()
}

func (worker *LogProcessingWorker) flushChunk() {
	for i, output := range worker.outputList {
		maybeChunk := output.FlushBuffer()
		if maybeChunk != nil {
			worker.procCounter.CountChunk(i, maybeChunk)
			output.AcceptChunk(*maybeChunk)
		}
	}
}
