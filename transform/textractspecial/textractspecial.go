// Package textractspecial provides 'extractHead' and 'extractTail' transforms, using prefix+wildcard+postfix for fast
// field extraction of simple cases, e.g. ".log.[0-9A-Z]" for filename "task.log.540001A".
package textractspecial

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for extractSpecialTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
	Pattern        string `yaml:"pattern"`
	MaxLength      int    `yaml:"maxLen"`
	DestKey        string `yaml:"destKey"`
}

type extractSpecialTransform struct {
	schema      base.LogSchema
	srcLocator  base.LogFieldLocator
	extractor   stringExtractor
	destLocator base.LogFieldLocator
}

// NewTransform creates extractSpecialTransform
func (c *Config) NewTransform(schema base.LogSchema, _ logger.Logger, _ base.LogCustomCounterRegistry) base.LogTransform {
	ex, err := newStringExtractorSimple(c.getPosition(), c.Pattern, c.MaxLength)
	if err != nil {
		panic(err)
	}
	return &extractSpecialTransform{
		schema:      schema,
		srcLocator:  schema.MustCreateFieldLocator(c.Key),
		extractor:   ex,
		destLocator: schema.MustCreateFieldLocator(c.DestKey),
	}
}

// VerifyConfig verifies extractSpecialTransform config
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
	if _, err := splitPattern(c.Pattern); err != nil {
		return fmt.Errorf(".pattern is invalid: %w", err)
	}
	if c.MaxLength <= 0 {
		return fmt.Errorf(".maxLength must larger than zero: %d", c.MaxLength)
	}
	if len(c.DestKey) == 0 {
		return fmt.Errorf(".destKey is unspecified")
	}
	if _, err := schema.CreateFieldLocator(c.DestKey); err != nil {
		return fmt.Errorf(".destKey '%s' is invalid: %w", c.DestKey, err)
	}
	return nil
}

func (c *Config) getPosition() stringExtractorPosition {
	switch c.Type {
	case "extractHead":
		return extractFromStart
	case "extractTail":
		return extractFromEnd
	default:
		panic(fmt.Sprintf("unsupported position type '%s'", c.Type))
	}
}

func (tf *extractSpecialTransform) Transform(record *base.LogRecord) base.FilterResult {
	fields := record.Fields
	value := tf.srcLocator.Get(fields)
	if len(value) == 0 {
		// TODO: metrics or logging?
		return base.PASS
	}
	extracted, valurRemaining := tf.extractor.Extract(value)
	if len(valurRemaining) != len(value) {
		tf.srcLocator.Set(fields, valurRemaining)
		tf.destLocator.Set(fields, extracted)
	}
	return base.PASS
}
