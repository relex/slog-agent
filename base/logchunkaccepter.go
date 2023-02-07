package base

// LogChunkAccepter is a function which accepts completed and loaded chunks for buffering or saving
type LogChunkAccepter func(chunk LogChunk)
