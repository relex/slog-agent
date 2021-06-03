// Package tredactemail provides 'redactEmail' transform to mask email addresses
package tredactemail

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for redactEmailTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
	Label          string `yaml:"label"`
}

type redactEmailTransform struct {
	keyLocator base.LogFieldLocator
	counter    func(length int)
}

// NewTransform creates redactEmailTransform
func (cfg *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	tf := &redactEmailTransform{
		keyLocator: schema.MustCreateFieldLocator(cfg.Key),
		counter:    customCounterRegistry.RegisterCustomCounter(cfg.Label),
	}
	return tf
}

// VerifyConfig verifies redactEmailTransform config
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if len(cfg.Key) == 0 {
		return fmt.Errorf(".key is unspecified")
	}
	if _, err := schema.CreateFieldLocator(cfg.Key); err != nil {
		return fmt.Errorf(".key '%s' is invalid: %w", cfg.Key, err)
	}
	if len(cfg.Label) == 0 {
		return fmt.Errorf(".label is unspecified")
	}
	return nil
}

func (tf *redactEmailTransform) Transform(record *base.LogRecord) base.FilterResult {
	value := tf.keyLocator.Get(record.Fields)
	if len(value) == 0 {
		return base.PASS
	}
	first := redactEmailFindFirst(value)
	if first == -1 {
		return base.PASS
	}
	newValue, numRedacted := redactEmail1(value, first)
	if numRedacted > 0 {
		tf.keyLocator.Set(record.Fields, newValue)
		tf.counter(record.RawLength)
	}
	return base.PASS
}
