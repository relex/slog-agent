package hybridbuffer

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter"
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
	pendingChunks              promexporter.RWGauge
	inputChunksTotalTransient  promexporter.RWCounter
	inputChunksTotalPersistent promexporter.RWCounter
	consumedChunksTotal        promexporter.RWCounter
	leftoverChunksTotal        promexporter.RWCounter
	droppedChunksTotal         promexporter.RWCounter
}

func newChunkManager(parentLogger logger.Logger, operator chunkOperator, metricFactory *base.MetricFactory, sendAllAtEnd bool) chunkManager {
	inputChunksTotal := metricFactory.AddOrGetCounterVec("input_chunks_total", "Numbers of input chunks", []string{"state"}, nil)
	return chunkManager{
		logger:       parentLogger.WithField(defs.LabelPart, "ChunkManager"),
		operator:     operator,
		sendAllAtEnd: sendAllAtEnd || !operator.HasDir(),
		metrics: chunkManagerMetrics{
			pendingChunks:              metricFactory.AddOrGetGauge("pending_chunks", "Numbers of pending chunks in buffer or output phase", nil, nil),
			inputChunksTotalTransient:  inputChunksTotal.WithLabelValues("transient"),
			inputChunksTotalPersistent: inputChunksTotal.WithLabelValues("persistent"),
			consumedChunksTotal:        metricFactory.AddOrGetCounter("consumed_chunks_total", "Numbers of output chunks consumed / forwarded", nil, nil),
			leftoverChunksTotal:        metricFactory.AddOrGetCounter("leftover_chunks_total", "Numbers of output chunks left for next start", nil, nil),
			droppedChunksTotal:         metricFactory.AddOrGetCounter("dropped_chunks_total", "Numbers of dropped chunks", nil, nil),
		},
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

func (man *chunkManager) OnChunkInput(chunk base.LogChunk, loaded bool) {
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

func (man *chunkManager) OnChunkDropped(chunk base.LogChunk) {
	man.operator.OnChunkDropped(chunk)
	man.metrics.droppedChunksTotal.Inc()
	man.metrics.pendingChunks.Dec()
}

func (man *chunkManager) WaitPendingChunks() bool {
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
	if totalNum > 0 {
		man.logger.Warnf("remained chunks on disk: num=%d", totalNum)
	}
	man.operator.Close()
}
