// Package texclude provides 'exclude' transform, which excludes matched log records by percentage
package texclude

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bmatch"
)

// Config for excludeTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Match          bmatch.LogMatcherConfig `yaml:"match"`
	Percentage     int                     `yaml:"percentage"`
	ExclusionLabel string                  `yaml:"exclusionLabel"`
	RetentionLabel string                  `yaml:"retentionLabel"`
}

type excludeTransform struct {
	matcher        bmatch.LogMatcher
	countExclusion func(length int)
	countRetention func(length int)
	targetRate     int64
	totalMatched   int64
	totalDropped   int64
}

// NewTransform creates excludeTransform
func (cfg *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	tf := &excludeTransform{
		matcher:        cfg.Match.NewMatcher(schema),
		countExclusion: customCounterRegistry.RegisterCustomCounter(cfg.ExclusionLabel),
		countRetention: customCounterRegistry.RegisterCustomCounter(cfg.RetentionLabel),
		targetRate:     int64(cfg.Percentage),
		totalMatched:   0,
		totalDropped:   0,
	}
	return tf
}

// VerifyConfig verifies excludeTransform config
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if len(cfg.Match) == 0 {
		return fmt.Errorf(".match is empty")
	}
	if err := cfg.Match.VerifyConfig(schema); err != nil {
		return fmt.Errorf(".match: %w", err)
	}
	if cfg.Percentage < 0 || cfg.Percentage > 100 {
		return fmt.Errorf(".percentage must be between 0 and 100: %d", cfg.Percentage)
	}
	if len(cfg.ExclusionLabel) == 0 {
		return fmt.Errorf(".exclusionLabel is unspecified")
	}
	if len(cfg.RetentionLabel) == 0 {
		return fmt.Errorf(".retentionLabel is unspecified")
	}
	return nil
}

func (tf *excludeTransform) Transform(record *base.LogRecord) base.FilterResult {
	if !tf.matcher.Match(record) {
		return base.PASS
	}

	if tf.totalMatched > 0 && 100*tf.totalDropped/tf.totalMatched < tf.targetRate {
		tf.totalMatched++
		tf.totalDropped++
		tf.countExclusion(record.RawLength)
		return base.DROP
	}
	tf.totalMatched++
	tf.countRetention(record.RawLength)
	return base.PASS
}
