package bconfig

import (
	"fmt"

	"github.com/relex/slog-agent/base"
)

// OutputBufferConfig configures buffer and output settings

type OutputBufferConfig struct {
	BaseConfig
	Name         string                          `yaml:"name"`
	BufferConfig ConfigHolder[ChunkBufferConfig] `yaml:"buffer"`
	OutputConfig ConfigHolder[LogOutputConfig]   `yaml:"output"`
}

func (cfg OutputBufferConfig) VerifyConfig(schema base.LogSchema) error {
	if err := cfg.BufferConfig.Value.VerifyConfig(); err != nil {
		return fmt.Errorf("buffer config validation error: %w", err)
	}

	if err := cfg.OutputConfig.Value.VerifyConfig(schema); err != nil {
		return fmt.Errorf("output config validation error: %w", err)
	}

	return nil
}
