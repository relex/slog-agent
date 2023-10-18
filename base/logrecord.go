package base

import (
	"time"

	"github.com/relex/slog-agent/util"
)

// LogRecord defines the structure of log record before it's finalized for forwarding.
type LogRecord struct {
	Fields    LogFields // Field values by index and empty string if unset. The string values inside are temporary and only valid until record is released.
	RawLength int       // Input length or approximated length of entire record, for statistics
	Timestamp time.Time // Timestamp, might be zero until processed by a LogTransform
	Unescaped bool      // Whether the main message field has been un-escaped. Multi-line logs start with true.
	Spam      bool
	_backbuf  *[]byte   // Backing buffer where initial field values come from, nil if buffer pooling isn't used
	_refCount int       // reference count, + outputs_length for new, -1 for release (back to pool)
}

// LogFields represents named fields in LogRecord, to be used with LogSchema.
//
// Fields are by default empty strings and empty fields are the same as missing fields, which should be excluded from output.
//
// Some fields at the end of this slice may be reserved by schema.MaxFields and they shouldn't be processed.
type LogFields []util.MutableString
