// Package tif provides 'if' transform, performing optional steps if the given conditions are satisfied
package tif

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bmatch"
	"github.com/relex/slog-agent/base/bsupport"
)

// Config for ifTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Match          bmatch.LogMatcherConfig            `yaml:"match"`
	Then           []bconfig.LogTransformConfigHolder `yaml:"then"`
}

type ifTransform struct {
	matcher   bmatch.LogMatcher
	thenSteps []base.LogTransformFunc
}

// NewTransform creates ifTransform
func (c *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	return &ifTransform{
		matcher:   c.Match.NewMatcher(schema),
		thenSteps: bsupport.NewTransformsFromConfig(c.Then, schema, parentLogger, customCounterRegistry),
	}
}

// VerifyConfig verifies ifTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Match) == 0 {
		return fmt.Errorf(".match is empty")
	}
	if err := c.Match.VerifyConfig(schema); err != nil {
		return fmt.Errorf(".match: %w", err)
	}
	if len(c.Then) == 0 {
		return fmt.Errorf(".then is empty")
	}
	if err := bsupport.VerifyTransformConfigs(c.Then, schema, ".then"); err != nil {
		return err
	}
	return nil
}

func (tf *ifTransform) Transform(record *base.LogRecord) base.FilterResult {
	if tf.matcher.Match(record) {
		return bsupport.RunTransforms(record, tf.thenSteps)
	}
	return base.PASS
}
