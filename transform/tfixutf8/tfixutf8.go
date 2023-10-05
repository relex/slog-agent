// Package tfixutf8 provides 'fixUTF8' transform to clean up invalid UTF-8 values.
package tfixutf8

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for fixUTF8Transform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
}

type fixUTF8Transform struct {
	keyLocator base.LogFieldLocator
}

// NewTransform creates fixUTF8Transform
func (cfg *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	tf := &fixUTF8Transform{
		keyLocator: schema.MustCreateFieldLocator(cfg.Key),
	}
	return tf
}

// VerifyConfig verifies fixUTF8Transform config
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if len(cfg.Key) == 0 {
		return fmt.Errorf(".key is unspecified")
	}
	if _, err := schema.CreateFieldLocator(cfg.Key); err != nil {
		return fmt.Errorf(".key '%s' is invalid: %w", cfg.Key, err)
	}
	return nil
}

func (tf *fixUTF8Transform) Transform(record *base.LogRecord) base.FilterResult {
	value := tf.keyLocator.Get(record.Fields)
	if len(value) == 0 {
		return base.PASS
	}
	
	return base.PASS
}
