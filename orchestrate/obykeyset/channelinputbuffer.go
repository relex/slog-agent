package obykeyset

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// channelInputBuffer buffers input logs before flushing them into a channel
//
// A channel can have multiple buffers
//
// A buffer is NOT thread-safe and it should be created for each of gorouting sending logs to a channel
type channelInputBuffer struct {
	Channel       chan<- []*base.LogRecord // point to channel in global map
	PendingLogs   []*base.LogRecord        // locally-buffered log records to be sent to channel
	PendingBytes  int
	LastFlushTime time.Time
}

func newInputBufferForChannel(ch chan<- []*base.LogRecord) *channelInputBuffer {
	return &channelInputBuffer{
		Channel:       ch,
		PendingLogs:   make([]*base.LogRecord, 0, defs.IntermediateBufferMaxNumLogs),
		PendingBytes:  0,
		LastFlushTime: time.Now(),
	}
}

// Append appends logs to buffer and checks whether flushing should be triggered
func (cache *channelInputBuffer) Append(record *base.LogRecord) bool { // xx:inline
	cache.PendingLogs = append(cache.PendingLogs, record)
	cache.PendingBytes += record.RawLength
	if cache.PendingBytes >= defs.IntermediateBufferMaxTotalBytes ||
		len(cache.PendingLogs) >= defs.IntermediateBufferMaxNumLogs {
		return true
	}
	return false
}

// Flush flushes all pending logs to the channel
func (cache *channelInputBuffer) Flush(now time.Time, sendTimeout *time.Timer, log logger.Logger, loggingKey interface{}) {
	pendingLogs := cache.PendingLogs
	reusableLogBuffer := bsupport.CopyLogBuffer(pendingLogs)
	cache.PendingLogs = pendingLogs[:0]
	cache.PendingBytes = 0
	cache.LastFlushTime = now

	select {
	case cache.Channel <- reusableLogBuffer:
		// TODO: update metrics
	case <-sendTimeout.C:
		log.Errorf("BUG: timeout flushing: %d records for %s. stack=%s", len(reusableLogBuffer), loggingKey, util.Stack())
	}
}
