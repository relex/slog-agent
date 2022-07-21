package base

import (
	"time"
)

// LogRecord defines the structure of log record before it's finalized for forwarding
type LogRecord struct {
	Timestamp time.Time
	_backbuf  *[]byte
	Fields    LogFields
	RawLength int
	_refCount int32
	Unescaped bool
}

// LogFields represents named fields in LogRecord, to be used with LogSchema
//
// Fields are by default empty strings and empty fields are the same as missing fields, which should be excluded from output
//
// Some fields at the end of this slice may be reserved by schema.MaxFields and they shouldn't be processed
type LogFields []string
