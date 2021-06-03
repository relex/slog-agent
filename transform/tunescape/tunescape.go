// Package tunescape provides 'unescape' transform, which handles custom escape characters like those in JSON strings.
//
// The transform equals to 'unescape' rewriter and should be avoided if possible to improve performance for large
// messages/fields.
package tunescape

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
)

// Config for unescapeTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
}

// unescapeTransform unescapes special characters like those in JSON strings
type unescapeTransform struct {
	keyLocator base.LogFieldLocator
}

var unescaper = bsupport.NewSyslogUnescaper()

// NewTransform creates unescapeTransform
func (c *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	return &unescapeTransform{
		keyLocator: schema.MustCreateFieldLocator(c.Key),
	}
}

// VerifyConfig verifies unescapeTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Key) == 0 {
		return fmt.Errorf(".key is unspecified")
	}
	if _, err := schema.CreateFieldLocator(c.Key); err != nil {
		return fmt.Errorf(".key '%s' is invalid: %w", c.Key, err)
	}
	return nil
}

func (tf *unescapeTransform) Transform(record *base.LogRecord) base.FilterResult {
	if record.Unescaped {
		return base.PASS
	}
	record.Unescaped = true
	value := tf.keyLocator.Get(record.Fields)
	if len(value) == 0 {
		return base.PASS
	}
	first := unescaper.FindFirst(value)
	if first == -1 {
		return base.PASS
	}
	result := unescaper.RunFromFirst(value, first)
	tf.keyLocator.Set(record.Fields, result)
	return base.PASS
}
