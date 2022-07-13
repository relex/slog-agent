package bconfig

// OutputBufferConfig configures buffer and output settings

type OutputBufferConfig struct {
	BaseConfig
	Name         string                          `yaml:"name"`
	BufferConfig ConfigHolder[ChunkBufferConfig] `yaml:"buffer"`
	OutputConfig ConfigHolder[LogOutputConfig]   `yaml:"output"`
}
