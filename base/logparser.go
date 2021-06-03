package base

import (
	"time"
)

// LogParser parses incoming logs from LogListener to structured records one by one
type LogParser interface {
	Parse(input []byte, timestamp time.Time) *LogRecord
}
