// Package tdelfields provides 'delFields' transform which removes (empties) fields from log records
package tdelfields

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for delFieldsTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Keys           []string `yaml:"keys"`
}

type delFieldsTransform struct {
	locators []base.LogFieldLocator
}

// NewTransform creates delFieldsTransform
func (c *Config) NewTransform(schema base.LogSchema, _ logger.Logger, _ base.LogCustomCounterRegistry) base.LogTransform {
	locators := make([]base.LogFieldLocator, 0, len(c.Keys))
	for _, key := range c.Keys {
		locators = append(locators, schema.MustCreateFieldLocator(key))
	}
	return &delFieldsTransform{
		locators: locators,
	}
}

// VerifyConfig verifies delFieldsTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Keys) == 0 {
		return fmt.Errorf(".keys is empty")
	}
	for _, key := range c.Keys {
		if _, err := schema.CreateFieldLocator(key); err != nil {
			return fmt.Errorf(".fields[%s] is invalid: %w", key, err)
		}
	}
	return nil
}

func (tf *delFieldsTransform) Transform(record *base.LogRecord) base.FilterResult {
	fields := record.Fields
	for _, loc := range tf.locators {
		loc.Del(fields)
	}
	return base.PASS
}
