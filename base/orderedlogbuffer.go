package base

import (
	"sync"
)

// OrderedLogBuffer carries log records and a lock for this batch to control order
// The lock may be reused for several buffers sent in sequence
type OrderedLogBuffer struct {
	Records  []*LogRecord
	Previous *sync.Mutex // mutex protecting previous batch to lock before the first result of this buffer can be sent
	Current  *sync.Mutex // mutex protecting current batch to unlock after the entire batch is finished (when Current changes)
	IsLast   bool
}
