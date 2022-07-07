package bsupport

import (
	"github.com/relex/slog-agent/base"
)

// CopyLogBuffer duplicates the given buffer of log records, for passing into channel
func CopyLogBuffer(slice []*base.LogRecord) []*base.LogRecord { // xx:inline
	// pooling of LogBuffer has been tried and resulted in no visible improvement
	return append([]*base.LogRecord(nil), slice...)
}

// SumLogRecordsLength calculates the total of RawLength of given records
func SumLogRecordsLength(records []*base.LogRecord) int {
	sum := 0
	for _, r := range records {
		sum += r.RawLength
	}
	return sum
}
