package obykeyset

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
	"github.com/relex/slog-agent/util/localcachedmap"
)

type globalPipelineChannelMap = localcachedmap.GlobalCachedMap[chan<- []*base.LogRecord, *pipelineChannelLocalBuffer]

type localPipelineChannelMap = localcachedmap.LocalCachedMap[chan<- []*base.LogRecord, *pipelineChannelLocalBuffer]

func closePipelineChannel(ch chan<- []*base.LogRecord) {
	close(ch)
}

// pipelineChannelLocalBuffer acts as goroutine-local cache and buffer for passing log batches to pipeline workers
//
// It collects pending logs and only sends logs to channel when certain limit is reached
type pipelineChannelLocalBuffer struct {
	Channel       chan<- []*base.LogRecord // point to channel in global map
	PendingLogs   []*base.LogRecord        // locally-buffered log records to be sent to channel
	PendingBytes  int
	LastFlushTime time.Time
}

func wrapPipelineChannelInLocalBuffer(ch chan<- []*base.LogRecord) *pipelineChannelLocalBuffer {
	return &pipelineChannelLocalBuffer{
		Channel:       ch,
		PendingLogs:   make([]*base.LogRecord, 0, defs.IntermediateBufferMaxNumLogs),
		PendingBytes:  0,
		LastFlushTime: time.Now(),
	}
}

// Append appends logs to buffer and checks whether flushing should be triggered
func (cache *pipelineChannelLocalBuffer) Append(record *base.LogRecord) bool { // xx:inline
	cache.PendingLogs = append(cache.PendingLogs, record)
	cache.PendingBytes += record.RawLength
	if cache.PendingBytes >= defs.IntermediateBufferMaxTotalBytes ||
		len(cache.PendingLogs) >= defs.IntermediateBufferMaxNumLogs {
		return true
	}
	return false
}

// Flush flushes all pending logs to the channel
func (cache *pipelineChannelLocalBuffer) Flush(now time.Time, sendTimeout *time.Timer, logger logger.Logger, loggingKey interface{}) {
	pendingLogs := cache.PendingLogs
	reusableLogBuffer := bsupport.CopyLogBuffer(pendingLogs)
	cache.PendingLogs = pendingLogs[:0]
	cache.PendingBytes = 0
	cache.LastFlushTime = now
	select {
	case cache.Channel <- reusableLogBuffer:
		// TODO: update metrics
		break
	case <-sendTimeout.C:
		logger.Errorf("BUG: timeout flushing: %d records for %s. stack=%s", len(reusableLogBuffer), loggingKey, util.Stack())
		break
	}
}
