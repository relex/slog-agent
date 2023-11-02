// Package ttruncate provides 'truncate' transform to truncate field values exceeding certain limit
package ttruncate

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/util"
)

// Config for truncateTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
	MaxLength      int    `yaml:"maxLen"`
	Suffix         string `yaml:"suffix"`
}

type truncateTransform struct {
	keyLocator base.LogFieldLocator
	maxLength  int
	suffix     string
}

// NewTransform creates truncateTransform
func (c *Config) NewTransform(schema base.LogSchema, _ logger.Logger, _ base.LogCustomCounterRegistry) base.LogTransform {
	return &truncateTransform{
		keyLocator: schema.MustCreateFieldLocator(c.Key),
		maxLength:  c.MaxLength,
		suffix:     c.Suffix,
	}
}

// VerifyConfig verifies truncateTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Key) == 0 {
		return fmt.Errorf(".key is unspecified")
	}
	if _, err := schema.CreateFieldLocator(c.Key); err != nil {
		return fmt.Errorf(".key is invalid: %w", err)
	}
	if c.MaxLength <= 0 {
		return fmt.Errorf(".maxLength must be larger than zero: %d", c.MaxLength)
	}
	if len(c.Suffix) == 0 {
		return fmt.Errorf(".suffix is unspecified")
	}
	return nil
}

func (tf *truncateTransform) Transform(record *base.LogRecord) base.FilterResult {
	value := tf.keyLocator.Get(record.Fields)
	if len(value) > tf.maxLength+len(tf.suffix) {
		valueB := util.BytesFromString(value)

		// truncate and clean up before the maxLength in case of UTF-8 sequences cut in the middle
		valueTrimmed := util.CleanUTF8(valueB[:tf.maxLength])
		// paste suffix at the truncated end - NOT the maxLength as the actual length could be smaller due to UTF-8 cleanup
		valueOverwritten := util.OverwriteNTruncate(valueB, len(valueTrimmed), tf.suffix)

		tf.keyLocator.Set(record.Fields, util.StringFromBytes(valueOverwritten))
	}
	return base.PASS
}
