package hybridbuffer

import (
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// bufferer is an intermediate buffer buf which saves log chunks to disk temporarily if needed.
// It consists of two main parts:
//  1. bufferer.Accept() can be called to push given chunks into the persistent queue and may unload chunks from memory
//     during the process.
//  2. An outputFeeder is launched in the background to read chunks from the persistent queue, load them from disk if
//     necessary, and then push loaded chunks into the in-memory queue, which is then read by the output worker.
//
// The persistent queue "inputChannel" stores all of the queued chunk data or their filenames if unloaded. There is no
// file scanning during the whole process and all chunks must be present in the persistent queue to be forwarded.
//
// As Go channels are fixed-sized, the maximum count of chunks allowed is limited by defs.BufferMaxNumChunksInQueue.
type bufferer struct {
	logger       logger.Logger
	queueDirPath string
	chunkMan     chunkManager
	feeder       outputFeeder
	inputChannel chan base.LogChunk        // internal; LogChunk.Data can be nil if unloaded / saved on disk
	inputClosed  *channels.SignalAwaitable // internal; to abort ongoing input processing
	metrics      bufferMetrics
}

type bufferMetrics struct {
	queuedChunksTransient  promext.RWGauge
	queuedChunksPersistent promext.RWGauge
}

// newBufferer creates a HybridBufferer.
//
// sendAllAtEnd: sends everything at shutdown and waits for all chunks to be confirmed by ChunkConsumerArgs.OnChunkConsumed.
// true for testing only. Same functionality is activated if queue directory cannot be accessed.
func newBufferer(parentLogger logger.Logger, rootPath string, bufferID string, matchChunkID func(string) bool,
	parentMetricCreator promreg.MetricCreator, maxStorageBytes int64, sendAllAtEnd bool,
) base.ChunkBufferer {
	bufLogger := parentLogger.WithField(defs.LabelComponent, "HybridBufferer")
	metricCreator := makeBufferMetricCreator(parentMetricCreator)
	queueDirPath := makeBufferQueueDir(bufLogger, rootPath, bufferID)

	chunkOp := newChunkOperator(bufLogger, queueDirPath, matchChunkID, metricCreator, maxStorageBytes)
	if chunkOp.HasDir() {
		bufLogger.Infof("use chunk saving dir: %s", queueDirPath)
	} else {
		bufLogger.Error("disable chunk saving due to IO error")
	}

	chunkMan := newChunkManager(bufLogger, chunkOp, metricCreator, sendAllAtEnd)

	inputChannel := make(chan base.LogChunk, defs.BufferMaxNumChunksInQueue)
	inputClosed := channels.NewSignalAwaitable()

	queuedChunks := metricCreator.AddOrGetGaugeVec("queued_chunks", "Numbers of currently queued chunks", []string{"state"}, nil)
	metrics := bufferMetrics{
		queuedChunksTransient:  queuedChunks.WithLabelValues("transient"),
		queuedChunksPersistent: queuedChunks.WithLabelValues("persistent"),
	}
	// reset gauges in case parentMetricCreator is reused, e.g. 2nd orchestrator for recovery mode
	metrics.queuedChunksTransient.Set(0)
	metrics.queuedChunksPersistent.Set(0)

	return &bufferer{
		logger:       bufLogger,
		queueDirPath: queueDirPath,
		chunkMan:     chunkMan,
		feeder:       newOutputFeeder(bufLogger, chunkMan, inputChannel, inputClosed, metrics),
		inputChannel: inputChannel,
		inputClosed:  inputClosed,
		metrics:      metrics,
	}
}

func (buf *bufferer) QueueDirPath() string {
	return buf.queueDirPath
}

func (buf *bufferer) Start() {
	buf.recoverExistingChunks()
	go buf.feeder.Run()
}

func (buf *bufferer) Stopped() channels.Awaitable {
	return buf.feeder.Stopped()
}

// RegisterNewConsumer creates the parameters for a new consumer.
// The args must be used by a newly launched consumer and call OnFinished at the end.
func (buf *bufferer) RegisterNewConsumer() base.ChunkConsumerArgs {
	return buf.feeder.RegisterNewConsumer()
}

// Accept accepts incoming chunks
func (buf *bufferer) Accept(chunk base.LogChunk) {
	// divide by 2 because channel length is not updated in time
	if buf.feeder.NumOutput() >= defs.BufferMaxNumChunksInMemory/2 {
		buf.logger.Debugf("unload chunk for queuing: id=%s len=%d", chunk.ID, len(chunk.Data))
		buf.chunkMan.OnChunkInput(false)
		if !buf.chunkMan.UnloadOrDropChunk(&chunk) {
			return
		}
	} else {
		buf.logger.Debugf("pass chunk to queue: id=%s len=%d", chunk.ID, len(chunk.Data))
		buf.chunkMan.OnChunkInput(true)
	}

	select {
	case buf.inputChannel <- chunk:
		if chunk.Data != nil {
			buf.metrics.queuedChunksTransient.Inc()
		} else {
			buf.metrics.queuedChunksPersistent.Inc()
		}
	default:
		buf.chunkMan.OnChunkDropped(chunk)
		if chunk.Data != nil {
			buf.logger.Warnf("queue overflow, drop loaded chunk: id=%s len=%d", chunk.ID, len(chunk.Data))
		} else {
			buf.logger.Warnf("queue overflow, drop unloaded chunk id=%s", chunk.ID)
		}
	}
}

// Destroy closes everything and saves all pending chunks
func (buf *bufferer) Destroy() {
	buf.logger.Infof("destroying: in=%d out=%d wait=%t", len(buf.inputChannel), buf.feeder.NumOutput(),
		buf.chunkMan.ShouldWaitPendingChunks())

	var runTimeout time.Duration
	if buf.chunkMan.WaitPendingChunks() {
		runTimeout = defs.IntermediateChannelTimeout * 2 // little to wait for there
	} else {
		runTimeout = defs.BufferShutDownTimeout + defs.IntermediateChannelTimeout // for saving chunks
	}

	close(buf.inputChannel)
	buf.inputClosed.Signal()

	buf.logger.Infof("waiting for feeder: in=%d out=%d", len(buf.inputChannel), buf.feeder.NumOutput())
	if !buf.feeder.Stopped().Wait(runTimeout) {
		buf.logger.Errorf("BUG: couldn't stop feeder in time. stack=%s", util.Stack())
	}
}

func (buf *bufferer) recoverExistingChunks() {
	numChunks := 0

RECOVERY_LOOP:
	for _, chunk := range buf.chunkMan.ScanChunks() {
		select {
		case buf.inputChannel <- chunk:
			buf.chunkMan.OnChunkInputRecovered(chunk)
			buf.metrics.queuedChunksPersistent.Inc()
			numChunks++
		default:
			buf.logger.Warnf("too many chunk files, skip id=%s", chunk.ID)
			break RECOVERY_LOOP
		}
	}
	buf.logger.Infof("recovered chunks count=%d", numChunks)
}
