// Package tdrop provides 'drop' transform, which drops all log records matching specific criteria
package tdrop

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bmatch"
)

// Config for dropTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Match          bmatch.LogMatcherConfig `yaml:"match"`
	Label          string                  `yaml:"label"`
}

type dropTransform struct {
	matcher bmatch.LogMatcher
	counter func(length int)
}

// NewTransform creates dropTransform
func (cfg *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	tf := &dropTransform{
		matcher: cfg.Match.NewMatcher(schema),
		counter: customCounterRegistry.RegisterCustomCounter(cfg.Label),
	}
	return tf
}

// VerifyConfig verifies dropTransform config
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if len(cfg.Match) == 0 {
		return fmt.Errorf(".match is empty")
	}
	if err := cfg.Match.VerifyConfig(schema); err != nil {
		return fmt.Errorf(".match: %w", err)
	}
	if len(cfg.Label) == 0 {
		return fmt.Errorf(".label is unspecified")
	}
	return nil
}

func (tf *dropTransform) Transform(record *base.LogRecord) base.FilterResult {
	if tf.matcher.Match(record) {
		tf.counter(record.RawLength)
		return base.DROP
	}
	return base.PASS
}
