package bsupport

import (
	"sync"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// OrderedLogProcessingWorker is like LogProcessingWorker but keeps buffer processing in order by locks
type OrderedLogProcessingWorker struct {
	PipelineWorkerBaseForOrderedLogBuffer
	deallocator   *base.LogAllocator
	procCounter   *base.LogProcessCounter
	transformList []base.LogTransformFunc
	serializer    base.LogSerializer
	chunkMaker    base.LogChunkMaker
	acceptChunk   base.LogChunkAccepter
	chunkBuffer   []base.LogChunk
	lastFlushTime time.Time
	lastPrevious  *sync.Mutex // mutex to acquire before sending any chunk, nil if acquired
	lastCurrent   *sync.Mutex // mutex to release when current batch is done
}

// NewOrderedLogProcessingWorker creates OrderedLogProcessingWorker
func NewOrderedLogProcessingWorker(parentLogger logger.Logger,
	input <-chan base.OrderedLogBuffer, deallocator *base.LogAllocator, procCounter *base.LogProcessCounter,
	transforms []base.LogTransformFunc, serializer base.LogSerializer, chunkMaker base.LogChunkMaker,
	acceptChunk base.LogChunkAccepter) *OrderedLogProcessingWorker {
	worker := &OrderedLogProcessingWorker{
		PipelineWorkerBaseForOrderedLogBuffer: NewPipelineWorkerBaseForOrderedLogBuffer(
			parentLogger.WithField(defs.LabelComponent, "OrderedLogProcessingWorker"),
			input,
		),
		deallocator:   deallocator,
		procCounter:   procCounter,
		transformList: transforms,
		serializer:    serializer,
		chunkMaker:    chunkMaker,
		acceptChunk:   acceptChunk,
		chunkBuffer:   make([]base.LogChunk, 0, 10),
		lastFlushTime: time.Now(),
		lastPrevious:  nil,
		lastCurrent:   nil,
	}
	worker.InitInternal(worker.onInput, worker.onTick, worker.onStop)
	return worker
}

func (worker *OrderedLogProcessingWorker) onInput(buffer base.OrderedLogBuffer, timeout <-chan time.Time) {
	if buffer.Current != worker.lastCurrent {
		worker.finishCurrentBatch(timeout)
		worker.lastCurrent = buffer.Current
		worker.lastPrevious = buffer.Previous
	}
	if len(buffer.Records) == 0 {
		return
	}
	for _, record := range buffer.Records {
		icounter := worker.procCounter.SelectInputCounter(record)
		if RunTransforms(record, worker.transformList) == base.DROP {
			icounter.CountRecordDrop(record)
			worker.deallocator.Release(record)
			continue
		}
		icounter.CountRecordPass(record)
		stream := worker.serializer.SerializeRecord(record)
		worker.procCounter.CountStream(stream)
		maybeChunk := worker.chunkMaker.WriteStream(stream)
		if maybeChunk != nil {
			worker.onChunkMade(*maybeChunk, timeout)
		}
	}
	if buffer.IsLast {
		worker.finishCurrentBatch(timeout)
	}
}

func (worker *OrderedLogProcessingWorker) onTick(timeout <-chan time.Time) {
	// flush buffered streams as a chunk if X seconds have passed
	if time.Since(worker.lastFlushTime) < defs.IntermediateFlushInterval {
		return
	}
	worker.flushChunk(timeout)
	worker.procCounter.UpdateMetrics()
}

func (worker *OrderedLogProcessingWorker) onStop(timeout <-chan time.Time) {
	worker.finishCurrentBatch(timeout)
	worker.procCounter.UpdateMetrics()
}

func (worker *OrderedLogProcessingWorker) finishCurrentBatch(timeout <-chan time.Time) {
	if worker.lastCurrent != nil {
		worker.flushChunk(timeout)
		if worker.lastPrevious != nil {
			worker.lastPrevious.Lock()
			worker.lastPrevious = nil
		}
		for _, chunk := range worker.chunkBuffer {
			worker.acceptChunk(chunk, timeout)
		}
		worker.chunkBuffer = worker.chunkBuffer[:0]
		worker.lastCurrent.Unlock()

		worker.lastPrevious = nil
		worker.lastCurrent = nil
	} else if len(worker.chunkBuffer) > 0 {
		worker.Logger().Panicf("unprotected chunks in buffer: %d", len(worker.chunkBuffer))
	}
}

func (worker *OrderedLogProcessingWorker) flushChunk(timeout <-chan time.Time) {
	maybeChunk := worker.chunkMaker.FlushBuffer()
	if maybeChunk != nil {
		worker.onChunkMade(*maybeChunk, timeout)
	}
}

func (worker *OrderedLogProcessingWorker) onChunkMade(chunk base.LogChunk, timeout <-chan time.Time) {
	worker.procCounter.CountChunk(&chunk)
	// try to acquire 'lastPrevious' so we can push chunk to receiver
	if worker.lastPrevious != nil && util.TryLockMutex(worker.lastPrevious) {
		worker.lastPrevious = nil
	}
	// push chunk if 'lastPrevious' has been unlocked. If not, queue it and continue to process.
	if worker.lastPrevious == nil {
		worker.acceptChunk(chunk, timeout)
	} else {
		worker.chunkBuffer = append(worker.chunkBuffer, chunk)
	}
}
