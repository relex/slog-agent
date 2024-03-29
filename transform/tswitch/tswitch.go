// Package tswitch provides 'switch' transform which acts like C switch without fallthrough.
// Nothing would happen if all of the cases fail to match a record.
package tswitch

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bmatch"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/samber/lo"
)

// Config for switchTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Cases          []CaseConfig `yaml:"cases"`
}

// CaseConfig for switchCase
type CaseConfig struct {
	Match bmatch.LogMatcherConfig            `yaml:"match"`
	Then  []bconfig.LogTransformConfigHolder `yaml:"then"`
}

type switchTransform struct {
	cases []switchCase
}

// switchCase acts like C switch "case", with nested cases and optional then steps if matched
type switchCase struct {
	matcher bmatch.LogMatcher
	then    []base.LogTransformFunc
}

type switchCaseResult bool

// NewTransform creates switchTransform
func (c *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	return &switchTransform{
		cases: lo.Map(c.Cases, func(caseConfig CaseConfig, _ int) switchCase {
			return caseConfig.newCase(schema, parentLogger, customCounterRegistry)
		}),
	}
}

// VerifyConfig verifies switchTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Cases) == 0 {
		return fmt.Errorf(".cases is empty")
	}
	for i, cas := range c.Cases {
		err := cas.verify(schema)
		if err != nil {
			return fmt.Errorf(".cases[%d]: %w", i, err)
		}
	}
	return nil
}

func (c *CaseConfig) newCase(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) switchCase {
	return switchCase{
		matcher: c.Match.NewMatcher(schema),
		then:    bsupport.NewTransformsFromConfig(c.Then, schema, parentLogger, customCounterRegistry),
	}
}

func (c *CaseConfig) verify(schema base.LogSchema) error {
	if len(c.Match) == 0 {
		return fmt.Errorf(".match is empty")
	}
	if err := c.Match.VerifyConfig(schema); err != nil {
		return fmt.Errorf(".match: %w", err)
	}
	if len(c.Then) == 0 {
		return fmt.Errorf(".then is empty")
	}

	return bsupport.VerifyTransformConfigs(c.Then, schema, ".then")
}

func (tf *switchTransform) Transform(record *base.LogRecord) base.FilterResult {
	for _, c := range tf.cases {
		matched, status := c.apply(record)
		if matched {
			return status
		}
	}
	return base.PASS
}

func (sc *switchCase) apply(record *base.LogRecord) (switchCaseResult, base.FilterResult) {
	if !sc.matcher.Match(record) {
		return false, base.PASS
	}
	return true, bsupport.RunTransforms(record, sc.then)
}
