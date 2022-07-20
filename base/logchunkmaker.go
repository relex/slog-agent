package base

// LogChunkMaker packages serialized log streams into certain chunk format for storage or transport, e.g. fluentd forward message
// One chunk may come from more than one stream joined together
type LogChunkMaker interface {

	// WriteStream packages the given log stream and optionally returns a completed chunk
	WriteStream(stream LogStream) *LogChunk

	// FlushBuffer packages buffered log stream(s) into a chunk if there is any
	FlushBuffer() *LogChunk
}
