// Package tblock provides 'block' transform, which groups child transform steps
package tblock

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
)

// Config for blockTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Steps          []bconfig.LogTransformConfigHolder `yaml:"steps"`
}

type blockTransform struct {
	steps []base.LogTransformFunc
}

// NewTransform creates blockTransform
func (c *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	return &blockTransform{
		steps: bsupport.NewTransformsFromConfig(c.Steps, schema, parentLogger, customCounterRegistry),
	}
}

// VerifyConfig verifies blockTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Steps) == 0 {
		return fmt.Errorf(".steps is empty")
	}
	return bsupport.VerifyTransformConfigs(c.Steps, schema, "steps")
}

func (tf *blockTransform) Transform(record *base.LogRecord) base.FilterResult {
	return bsupport.RunTransforms(record, tf.steps)
}
