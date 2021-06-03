// Package tmapvalue provides 'mapValue' transform, providing one-to-one mapping on a field value.
// For example, translating syslog severities to log4j2 levels.
package tmapvalue

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for mapValueTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string            `yaml:"key"`
	Mapping        map[string]string `yaml:"mapping"`
	DefaultValue   string            `yaml:"default"`
}

type mapValueTransform struct {
	keyLocator   base.LogFieldLocator
	mapping      map[string]string
	defaultValue string
}

// NewTransform creates mapValueTransform
func (c *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	return &mapValueTransform{
		keyLocator:   schema.MustCreateFieldLocator(c.Key),
		mapping:      c.Mapping,
		defaultValue: c.DefaultValue,
	}
}

// VerifyConfig verifies mapValueTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Key) == 0 {
		return fmt.Errorf(".key is unspecified")
	}
	if _, err := schema.CreateFieldLocator(c.Key); err != nil {
		return fmt.Errorf(".key '%s' is invalid: %w", c.Key, err)
	}
	if len(c.Mapping) == 0 {
		return fmt.Errorf(".mapping is empty")
	}
	return nil
}

func (tf *mapValueTransform) Transform(record *base.LogRecord) base.FilterResult {
	oldValue := tf.keyLocator.Get(record.Fields)
	if len(oldValue) == 0 {
		return base.PASS
	}
	newValue, ok := tf.mapping[oldValue]
	if !ok {
		newValue = tf.defaultValue
	}
	tf.keyLocator.Set(record.Fields, newValue)
	return base.PASS
}
