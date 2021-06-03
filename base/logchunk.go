package base

import (
	"fmt"
)

// LogChunk represents a chunk of log records serialized and ready for storage or transport as its own unit
type LogChunk struct {
	ID    string // Unique ID of this chunk, may be used as filename
	Data  []byte // Actual data of this chunk, transformed from LogStream
	Saved bool   // true if saved on disk already
}

// LogChunkInfo contains information that can be extracted from a chunk binary for test purposes
type LogChunkInfo struct {
	Tag        string
	NumRecords int
}

func (chunk LogChunk) String() string {
	switch {
	case chunk.Data == nil && chunk.Saved:
		return fmt.Sprintf("id=%s unloaded", chunk.ID)
	case chunk.Saved:
		return fmt.Sprintf("id=%s len=%d saved", chunk.ID, len(chunk.Data))
	default:
		return fmt.Sprintf("id=%s len=%d", chunk.ID, len(chunk.Data))
	}
}
