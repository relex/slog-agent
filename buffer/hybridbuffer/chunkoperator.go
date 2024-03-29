package hybridbuffer

import (
	"io"
	"os"
	"sort"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
)

type chunkOperator struct {
	logger        logger.Logger
	maybeDir      *os.File
	matchChunkID  func(string) bool // check whether a filename is a valid chunk ID (e.g. with proper suffix)
	metrics       chunkOperatorMetrics
	maxTotalBytes int64
}

// chunkOperatorMetrics tracks all low-level metrics.
// Both of persistentChunks and persistentChunkBytes only concern chunks in the agent, not necessarily all files on
// disk. For instance, unreadable files would be logged and then ignored, with their numbers removed from the metrics.
type chunkOperatorMetrics struct {
	persistentChunks     promext.RWGauge
	persistentChunkBytes promext.RWGauge
	ioErrorsTotal        promext.RWCounter
}

func newChunkOperator(parentLogger logger.Logger, path string, matchChunkID func(string) bool,
	metricCreator promreg.MetricCreator, maxTotalBytes int64,
) chunkOperator {
	ologger := parentLogger

	metrics := chunkOperatorMetrics{
		persistentChunks:     metricCreator.AddOrGetGauge("persistent_chunks", "Numbers of currently persistent chunks, including chunks being sent but not yet acknowledged.", nil, nil),
		persistentChunkBytes: metricCreator.AddOrGetGauge("persistent_chunk_bytes", "Bytes of currently persistent chunks, including chunks being sent but not yet acknowledged", nil, nil),
		ioErrorsTotal:        metricCreator.AddOrGetCounter("io_errors_total", "Numbers of I/O errors for chunk operations", nil, nil),
	}
	// reset gauges in case metricCreator is reused, e.g. 2nd orchestrator for recovery mode
	metrics.persistentChunks.Set(0)
	metrics.persistentChunkBytes.Set(0)

	maybeDir, oerr := os.Open(path)
	if oerr != nil {
		ologger.Errorf("error opening baseDir path=%s: %s", path, oerr.Error())
		maybeDir = nil
		metrics.ioErrorsTotal.Inc()
	}

	return chunkOperator{
		logger:        ologger,
		maybeDir:      maybeDir,
		matchChunkID:  matchChunkID,
		metrics:       metrics,
		maxTotalBytes: maxTotalBytes,
	}
}

func (op *chunkOperator) HasDir() bool {
	return op.maybeDir != nil
}

func (op *chunkOperator) CountExistingChunks() int {
	if op.maybeDir == nil {
		return 0
	}

	if _, serr := op.maybeDir.Seek(0, io.SeekStart); serr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error seeking directory: %s", serr.Error())
		return 0
	}

	fnames, derr := op.maybeDir.Readdirnames(0)
	if derr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error counting directory: %s", derr.Error())
		return 0
	}

	numChunks := 0
	for _, fn := range fnames {
		if op.matchChunkID(fn) {
			numChunks++
		}
	}
	return numChunks
}

func (op *chunkOperator) ScanExistingChunks() []base.LogChunk {
	if op.maybeDir == nil {
		return nil
	}

	// reset position to start or it would only list new files on subsequent calls
	if _, serr := op.maybeDir.Seek(0, io.SeekStart); serr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error seeking directory: %s", serr.Error())
	}

	fnames, derr := op.maybeDir.Readdirnames(0)
	if derr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error scanning directory: %s", derr.Error())
		return nil
	}
	sort.Strings(fnames)

	chunkList := make([]base.LogChunk, 0, len(fnames))
	for _, fn := range fnames {
		if fn == idFileName {
			continue
		}
		if !op.matchChunkID(fn) {
			op.logger.Warnf("skip unmatched chunk file id=%s", fn)
			continue
		}
		chunk := base.LogChunk{ID: fn, Data: nil, Saved: true}
		chunkList = append(chunkList, chunk)
	}
	return chunkList
}

func (op *chunkOperator) LoadChunk(chunkRef *base.LogChunk) bool {
	if chunkRef.Data != nil {
		return true
	}
	if !chunkRef.Saved {
		op.logger.Errorf("BUG: cannot load unsaved chunk id=%s. stack=%s", chunkRef.ID, util.Stack())
		return false
	}
	if op.maybeDir == nil {
		op.logger.Errorf("BUG: cannot load chunk id=%s with nil dir. stack=%s", chunkRef.ID, util.Stack())
		return false
	}

	data, rerr := util.ReadFileAt(op.maybeDir, chunkRef.ID)
	if rerr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error reading chunk id=%s: %s", chunkRef.ID, rerr.Error())
		return false
	}

	chunkRef.Data = data
	return true
}

func (op *chunkOperator) UnloadChunk(chunkRef *base.LogChunk) bool {
	if chunkRef.Saved {
		return true
	}
	if chunkRef.Data == nil {
		op.logger.Errorf("BUG: cannot unload nil chunk id=%s. stack=%s", chunkRef.ID, util.Stack())
		return false
	}
	if op.maybeDir == nil {
		// fail silently as expected
		return false
	}

	if op.metrics.persistentChunkBytes.Get()+int64(len(chunkRef.Data)) > op.maxTotalBytes {
		op.logger.Warnf("cannot write chunk file id=%s: space limit reached", chunkRef.ID)
		return false
	}

	if werr := util.WriteFileAt(op.maybeDir, chunkRef.ID, chunkRef.Data, 0o644); werr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error writing chunk id=%s: %s", chunkRef.ID, werr.Error())
		return false
	}

	op.metrics.persistentChunks.Inc()
	op.metrics.persistentChunkBytes.Add(int64(len(chunkRef.Data)))
	chunkRef.Data = nil
	chunkRef.Saved = true
	return true
}

func (op *chunkOperator) RemoveChunk(chunk base.LogChunk) {
	if !chunk.Saved {
		return
	}
	if op.maybeDir == nil {
		op.logger.Errorf("BUG: cannot remove chunk id=%s with nil dir. stack=%s", chunk.ID, util.Stack())
		return
	}

	if werr := util.UnlinkFileAt(op.maybeDir, chunk.ID); werr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error deleting chunk id=%s: %s", chunk.ID, werr.Error())
		return
	}

	op.metrics.persistentChunks.Dec()
	op.metrics.persistentChunkBytes.Sub(int64(len(chunk.Data)))
}

func (op *chunkOperator) OnChunkDropped(chunk base.LogChunk) {
	if !chunk.Saved {
		return
	}

	op.metrics.persistentChunks.Dec()
	op.metrics.persistentChunkBytes.Sub(int64(len(chunk.Data)))
}

func (op *chunkOperator) OnChunkRecovered(chunk base.LogChunk) {
	op.metrics.persistentChunks.Inc() // update before condition check

	if op.maybeDir == nil {
		op.logger.Errorf("BUG: cannot check size of recovered chunk id=%s with nil dir. stack=%s", chunk.ID, util.Stack())
		return
	}

	stat, serr := util.StatFileAt(op.maybeDir, chunk.ID)
	if serr != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error stating recovered chunk id=%s: %s", chunk.ID, serr.Error())
		return
	}

	op.metrics.persistentChunkBytes.Add(stat.Size)
}

func (op *chunkOperator) Close() {
	if op.maybeDir == nil {
		return
	}

	if err := op.maybeDir.Close(); err != nil {
		op.metrics.ioErrorsTotal.Inc()
		op.logger.Errorf("error closing dir: %s", err.Error())
	}
}
