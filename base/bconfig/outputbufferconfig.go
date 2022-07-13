package bconfig

// OutputBufferConfig configures buffer and output settings
type OutputBufferConfig struct {
	BufferConfig ChunkBufferConfig // Verified buffer config
	OutputConfig LogOutputConfig   // Verified output config
}
