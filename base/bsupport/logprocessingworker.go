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
	analyzer      base.LogAnalyzer
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
	analyzer base.LogAnalyzer, transforms []base.LogTransformFunc, outputInterfaces []OutputInterface,
) *LogProcessingWorker {
	worker := &LogProcessingWorker{
		PipelineWorkerBase: NewPipelineWorkerBase(
			parentLogger.WithField(defs.LabelComponent, "LogProcessingWorker"),
			input,
		),
		deallocator:   deallocator,
		procCounter:   procCounter,
		analyzer:      analyzer,
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
	worker.lastChunkTime = batch.Records[0].Timestamp

	releaseRecord := worker.deallocator.Release
	analyzingBatch := worker.analyzer.ShouldAnalyze(batch)
	if analyzingBatch {
		releaseRecord = func(record *base.LogRecord) {}
	}

	var numCleanRecords int
	var numCleanBytes int64

	for _, record := range batch.Records {
		icounter := worker.procCounter.SelectMetricKeySet(record)
		result := RunTransforms(record, worker.transformList)
		if !record.Spam {
			numCleanRecords++
			numCleanBytes += int64(record.RawLength)
		}
		if result == base.DROP {
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
		worker.analyzer.Analyze(batch, numCleanRecords, numCleanBytes)
		for _, record := range batch.Records {
			releaseRecord(record)
		}
	} else {
		worker.analyzer.TrackTraffic(numCleanRecords, numCleanBytes)
	}
}

func (worker *LogProcessingWorker) onTick() {
	worker.analyzer.Tick()

	// flush buffered streams only if one full second has passed
	if time.Since(worker.lastChunkTime) >= defs.IntermediateFlushInterval {
		worker.flushChunk()
	}
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
