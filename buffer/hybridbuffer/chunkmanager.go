package hybridbuffer

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

type chunkManager struct {
	logger       logger.Logger
	operator     chunkOperator
	sendAllAtEnd bool
	metrics      chunkManagerMetrics
}

type chunkManagerMetrics struct {
	pendingChunks              promext.RWGauge
	inputChunksTotalTransient  promext.RWCounter
	inputChunksTotalPersistent promext.RWCounter
	consumedChunksTotal        promext.RWCounter
	leftoverChunksTotal        promext.RWCounter
	droppedChunksTotal         promext.RWCounter
}

func newChunkManager(parentLogger logger.Logger, operator chunkOperator, metricCreator promreg.MetricCreator, sendAllAtEnd bool) chunkManager {
	inputChunksTotal := metricCreator.AddOrGetCounterVec("input_chunks_total", "Numbers of input chunks", []string{"state"}, nil)

	metrics := chunkManagerMetrics{
		pendingChunks:              metricCreator.AddOrGetGauge("pending_chunks", "Numbers of pending chunks in buffer or output phase", nil, nil),
		inputChunksTotalTransient:  inputChunksTotal.WithLabelValues("transient"),
		inputChunksTotalPersistent: inputChunksTotal.WithLabelValues("persistent"),
		consumedChunksTotal:        metricCreator.AddOrGetCounter("consumed_chunks_total", "Numbers of output chunks consumed / forwarded", nil, nil),
		leftoverChunksTotal:        metricCreator.AddOrGetCounter("leftover_chunks_total", "Numbers of output chunks left for next start", nil, nil),
		droppedChunksTotal:         metricCreator.AddOrGetCounter("dropped_chunks_total", "Numbers of dropped chunks", nil, nil),
	}
	// reset gauges in case metricCreator is reused, e.g. 2nd orchestrator for recovery mode
	metrics.pendingChunks.Set(0)

	return chunkManager{
		logger:       parentLogger.WithField(defs.LabelPart, "ChunkManager"),
		operator:     operator,
		sendAllAtEnd: sendAllAtEnd || !operator.HasDir(),
		metrics:      metrics,
	}
}

func (man *chunkManager) ScanChunks() []base.LogChunk {
	return man.operator.ScanExistingChunks()
}

func (man *chunkManager) LoadOrDropChunk(chunkRef *base.LogChunk) bool {
	if !man.operator.LoadChunk(chunkRef) {
		man.OnChunkDropped(*chunkRef)
		return false
	}
	return true
}

func (man *chunkManager) UnloadOrDropChunk(chunkRef *base.LogChunk) bool {
	if !man.operator.UnloadChunk(chunkRef) {
		man.OnChunkDropped(*chunkRef)
		return false
	}
	return true
}

func (man *chunkManager) OnChunkInput(loaded bool) {
	man.metrics.pendingChunks.Inc()
	if loaded {
		man.metrics.inputChunksTotalTransient.Inc()
	} else {
		man.metrics.inputChunksTotalPersistent.Inc()
	}
}

func (man *chunkManager) OnChunkInputRecovered(chunk base.LogChunk) {
	man.metrics.pendingChunks.Inc()
	man.metrics.inputChunksTotalPersistent.Inc()
	man.operator.OnChunkRecovered(chunk)
}

func (man *chunkManager) OnChunkConsumed(chunk base.LogChunk) {
	man.operator.RemoveChunk(chunk)
	man.metrics.pendingChunks.Dec()
	man.metrics.consumedChunksTotal.Inc()
}

func (man *chunkManager) OnChunkLeftover(chunk base.LogChunk) {
	man.logger.Debugf("save leftover id=%s len=%d", chunk.ID, len(chunk.Data))
	man.operator.UnloadChunk(&chunk)
	man.metrics.pendingChunks.Dec()
	man.metrics.leftoverChunksTotal.Inc()
}

func (man *chunkManager) OnChunkCorrupted(chunk base.LogChunk) {
	man.operator.RemoveChunk(chunk)
	man.metrics.pendingChunks.Dec()
	man.metrics.droppedChunksTotal.Inc()
}

func (man *chunkManager) OnChunkDropped(chunk base.LogChunk) {
	man.operator.OnChunkDropped(chunk)
	man.metrics.pendingChunks.Dec()
	man.metrics.droppedChunksTotal.Inc()
}

func (man *chunkManager) ShouldWaitPendingChunks() bool {
	return man.sendAllAtEnd
}

func (man *chunkManager) WaitPendingChunks() bool {
	// sendAllAtEnd should only be true in test or recovery mode (TBD)
	if man.sendAllAtEnd {
		man.logger.Infof("waiting for pending chunks: count=%d", man.metrics.pendingChunks.Get())
		if !man.metrics.pendingChunks.WaitForZero(defs.BufferShutDownTimeout) {
			man.logger.Errorf("failed to wait for pending chunks, count=%d", man.metrics.pendingChunks.Get())
		}
		return true
	}
	return false
}

func (man *chunkManager) Close() {
	totalNum := man.operator.CountExistingChunks()
	man.logger.Infof("remained chunks on disk: num=%d", totalNum)
	man.operator.Close()
}
