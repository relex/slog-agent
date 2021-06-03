package base

import (
	"sync"
	"time"

	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// LogAllocator allocates empty log records and backing buffers
// Local-cache with buffers of recycled logs has been tried and made minimal improvement
type LogAllocator struct {
	recordPool   *sync.Pool         // pool of *LogRecord
	backbufPools util.BytesPoolBy2n // pools of the backing buffers of LogRecord(s), i.e. pools of raw input copies
}

// NewLogAllocator creates LogAllocator linked to the given schema
func NewLogAllocator(schema LogSchema) *LogAllocator {
	numFields := len(schema.GetFieldNames())
	recordPool := &sync.Pool{}
	recordPool.New = func() interface{} {
		return newLogRecord(numFields)
	}
	return &LogAllocator{
		recordPool:   recordPool,
		backbufPools: util.NewBytesPoolBy2n(),
	}
}

func newLogRecord(numFields int) *LogRecord {
	return &LogRecord{
		Fields:    make(LogFields, numFields),
		RawLength: 0,
		Timestamp: time.Time{},
		Unescaped: false,
		_backbuf:  nil,
		_refCount: 0,
	}
}

// NewRecord creates new record of empty values
func (alloc *LogAllocator) NewRecord(input []byte) (*LogRecord, string) {
	record := alloc.recordPool.Get().(*LogRecord)
	record._refCount++
	if len(input) > defs.InputLogMinMessageBytesToPool {
		backbuf := alloc.backbufPools.Get(len(input))
		record._backbuf = backbuf
		n := copy(*backbuf, input)
		return record, util.StringFromBytes((*backbuf)[:n])
	}
	return record, util.DeepCopyStringFromBytes(input)
}

// Release releases this log record for recycling
func (alloc *LogAllocator) Release(record *LogRecord) {
	record._refCount--
	if record._refCount < 0 {
		panic(record)
	}
	for i := range record.Fields {
		record.Fields[i] = ""
	}
	record.RawLength = 0
	record.Timestamp = time.Time{}
	alloc.recycleRecord(record)
}

func (alloc *LogAllocator) recycleRecord(record *LogRecord) {
	if record._backbuf != nil {
		alloc.backbufPools.Put(record._backbuf)
		record._backbuf = nil
	}
	alloc.recordPool.Put(record)
}
