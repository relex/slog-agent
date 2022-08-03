// Package treplace provides 'replace' transform to performs replacements by regular expression on specified field.
// Both named and unnamed captures are supported in replacement, as in Regexp.Expand
//
// DO NOT use this transform in production.
package treplace

import (
	"fmt"
	"regexp"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for replaceTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
	Pattern        string `yaml:"pattern"`
	Replacement    string `yaml:"replacement"`
}

type replaceTransform struct {
	keyLocator  base.LogFieldLocator
	pattern     *regexp.Regexp
	replacement string
}

// NewTransform creates replaceTransform
func (c *Config) NewTransform(schema base.LogSchema, _ logger.Logger, _ base.LogCustomCounterRegistry) base.LogTransform {
	return &replaceTransform{
		keyLocator:  schema.MustCreateFieldLocator(c.Key),
		pattern:     regexp.MustCompile(c.Pattern),
		replacement: c.Replacement,
	}
}

// VerifyConfig verifies replaceTransform config
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
	_, err := regexp.Compile(c.Pattern)
	if err != nil {
		return fmt.Errorf(".pattern: %w", err)
	}
	return nil
}

func (tf *replaceTransform) Transform(record *base.LogRecord) base.FilterResult {
	value := tf.keyLocator.Get(record.Fields)
	if len(value) == 0 {
		return base.PASS
	}
	tf.keyLocator.Set(record.Fields, tf.pattern.ReplaceAllString(value, tf.replacement))
	return base.PASS
}
