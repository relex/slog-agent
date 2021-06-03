package obase

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
)

// PipelineChannel is an alias to write-only channel of log records
type PipelineChannel = chan<- []*base.LogRecord

// PipelineChannelLocalBuffer acts as goroutine/worker-local cache and buffer for writing to PipelineChannel
// It collects pending logs and only sends to channel when certain limit is reached
type PipelineChannelLocalBuffer struct {
	Channel       chan<- []*base.LogRecord // point to channel in global map
	PendingLogs   []*base.LogRecord        // locally-buffered log records to be sent to channel
	PendingBytes  int
	LastFlushTime time.Time
}

// NewPipelineChannelLocalBuffer creates a PipelineChannelLocalBuffer
func NewPipelineChannelLocalBuffer(ch chan<- []*base.LogRecord) *PipelineChannelLocalBuffer {
	return &PipelineChannelLocalBuffer{
		Channel:       ch,
		PendingLogs:   make([]*base.LogRecord, 0, defs.IntermediateBufferMaxNumLogs),
		PendingBytes:  0,
		LastFlushTime: time.Now(),
	}
}

// Append appends logs to buffer and checks whether flushing should be triggered
func (cache *PipelineChannelLocalBuffer) Append(record *base.LogRecord) bool { // xx:inline
	cache.PendingLogs = append(cache.PendingLogs, record)
	cache.PendingBytes += record.RawLength
	if cache.PendingBytes >= defs.IntermediateBufferMaxTotalBytes ||
		len(cache.PendingLogs) >= defs.IntermediateBufferMaxNumLogs {
		return true
	}
	return false
}

// Flush flushes all pending logs to the channel
func (cache *PipelineChannelLocalBuffer) Flush(now time.Time, sendTimeout *time.Timer, logger logger.Logger, loggingKey interface{}) {
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
		logger.Errorf("BUG: timeout flushing: %d records for %s", len(reusableLogBuffer), loggingKey)
		break
	}
}
