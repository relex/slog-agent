package base

import (
	"time"
)

// LogChunkAccepter is a function which accepts completed and loaded chunks for buffering or saving
type LogChunkAccepter func(chunk LogChunk, timeout <-chan time.Time)
