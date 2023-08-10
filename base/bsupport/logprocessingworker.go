package bsupport

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// LogProcessingWorker is a worker for transformation, serialization and chunk making
type LogProcessingWorker struct {
	PipelineWorkerBase[base.LogRecordBatch]
	deallocator   *base.LogAllocator
	procCounter   *base.LogProcessCounterSet
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
	input <-chan base.LogRecordBatch, deallocator *base.LogAllocator, procCounter *base.LogProcessCounterSet,
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

func (worker *LogProcessingWorker) onInput(batch base.LogRecordBatch) {
	if len(batch.Records) == 0 {
		return
	}

	releaseRecord := worker.deallocator.Release
	analyzingBatch := worker.shouldAnalyzeBatch(batch)
	if analyzingBatch {
		releaseRecord = func(record *base.LogRecord) {}
	}

	for _, record := range batch.Records {
		icounter := worker.procCounter.SelectMetricKeySet(record)
		if RunTransforms(record, worker.transformList) == base.DROP {
			icounter.CountRecordDrop(record)
			releaseRecord(record)
			continue
		}
		icounter.CountRecordPass(record)

		for i, output := range worker.outputList {
			// TODO:
			//if RunTransforms(record, output.transformList) == base.DROP {
			//	icounter.CountOutputFilter(i, record)
			//	releaseRecord(record)
			//	continue
			//}
			stream := output.SerializeRecord(record)
			worker.procCounter.CountStream(i, stream)
			maybeChunk := output.WriteStream(stream)
			if maybeChunk != nil {
				worker.procCounter.CountChunk(i, maybeChunk)
				output.AcceptChunk(*maybeChunk)
			}
		}
		releaseRecord(record)
	}

	if analyzingBatch {
		for _, record := range batch.Records {
			releaseRecord(record)
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

func (worker *LogProcessingWorker) shouldAnalyzeBatch(batch base.LogRecordBatch) bool {
	// TODO: determine whether to analyze this batch of log records
	return false
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
