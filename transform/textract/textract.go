// Package textract provides 'extract' transform, which parses specified field with regular expression and updates
// fields with named captures (overriding any existing value).
//
// Only the named captures from the first match are checked; Unnamed captures and subsequent matches are ignored.
//
// DO NOT use this transform in production.
package textract

import (
	"fmt"
	"regexp"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for extractTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
	Pattern        string `yaml:"pattern"`
}

type extractTransform struct {
	keyLocator          base.LogFieldLocator
	pattern             *regexp.Regexp
	subexpFieldLocators []base.LogFieldLocator
}

// NewTransform creates extractTransform
func (c *Config) NewTransform(schema base.LogSchema, _ logger.Logger, _ base.LogCustomCounterRegistry) base.LogTransform {
	re := regexp.MustCompile(c.Pattern)
	subexpSels := make([]base.LogFieldLocator, len(re.SubexpNames()))
	for i, name := range re.SubexpNames() {
		if len(name) == 0 {
			subexpSels[i] = base.MissingFieldLocator
			continue
		}
		subexpSels[i] = schema.MustCreateFieldLocator(name)
	}
	return &extractTransform{
		keyLocator:          schema.MustCreateFieldLocator(c.Key),
		pattern:             re,
		subexpFieldLocators: subexpSels,
	}
}

// VerifyConfig verifies extractTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Key) == 0 {
		return fmt.Errorf(".key is unspecified")
	}
	if _, err := schema.CreateFieldLocator(c.Key); err != nil {
		return fmt.Errorf(".key '%s' is invalid: %w", c.Key, err)
	}
	if len(c.Pattern) == 0 {
		return fmt.Errorf(".pattern is unspecified")
	}
	re, err := regexp.Compile(c.Pattern)
	if err != nil {
		return fmt.Errorf(".pattern: %w", err)
	}
	for _, name := range re.SubexpNames() {
		if _, err := schema.CreateFieldLocator(c.Key); err != nil {
			return fmt.Errorf(".pattern: named capture '%s' is invalid: %w", name, err)
		}
	}
	return nil
}

func (tf *extractTransform) Transform(record *base.LogRecord) base.FilterResult {
	fields := record.Fields
	value := tf.keyLocator.Get(fields)
	submatchIndexes := tf.pattern.FindStringSubmatchIndex(value)
	if submatchIndexes == nil {
		// TODO: metrics or logging?
		return base.PASS
	}
	for subexpIndex, locator := range tf.subexpFieldLocators {
		if locator == base.MissingFieldLocator {
			continue
		}
		start := submatchIndexes[2*subexpIndex]
		end := submatchIndexes[2*subexpIndex+1]
		if start < 0 || end < 0 {
			continue
		}
		locator.Set(fields, value[start:end])
	}
	return base.PASS
}
