// Package tdrop provides 'drop' transform, which drops all log records matching specific criteria
package tdrop

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bmatch"
)

// Config for dropTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Match          bmatch.LogMatcherConfig `yaml:"match"`
	Percentage     int                     `yaml:"percentage"`
	MetricLabel    string                  `yaml:"metricLabel"`
}

type dropTransform struct {
	matcher       bmatch.LogMatcher
	countDropped  func(length int)
	countRetained func(length int)
	targetRate    int64 // rate to drop matched logs, from 0 to 100 (percent)
	totalMatched  int64
	totalDropped  int64
}

// NewTransform creates dropTransform
func (cfg *Config) NewTransform(schema base.LogSchema, _ logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	tf := &dropTransform{
		matcher:       cfg.Match.NewMatcher(schema),
		countDropped:  customCounterRegistry.RegisterCustomCounter(cfg.MetricLabel),
		countRetained: nil,
		targetRate:    int64(cfg.Percentage),
		totalMatched:  0,
		totalDropped:  0,
	}

	if tf.targetRate < 100 {
		tf.countRetained = customCounterRegistry.RegisterCustomCounter("!" + cfg.MetricLabel)
	}

	return tf
}

// VerifyConfig verifies dropTransform config
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if len(cfg.Match) == 0 {
		return fmt.Errorf(".match is empty")
	}
	if err := cfg.Match.VerifyConfig(schema); err != nil {
		return fmt.Errorf(".match: %w", err)
	}
	if cfg.Percentage < 1 || cfg.Percentage > 100 {
		return fmt.Errorf(".percentage must be between 1 and 100: %d", cfg.Percentage)
	}
	if len(cfg.MetricLabel) == 0 {
		return fmt.Errorf(".metricLabel is unspecified")
	}
	return nil
}

func (tf *dropTransform) Transform(record *base.LogRecord) base.FilterResult {
	if !tf.matcher.Match(record) {
		return base.PASS
	}

	if tf.targetRate == 100 {
		tf.countDropped(record.RawLength)
		return base.DROP
	}

	if tf.totalMatched > 0 && 100*tf.totalDropped/tf.totalMatched < tf.targetRate {
		tf.totalMatched++
		tf.totalDropped++
		tf.countDropped(record.RawLength)
		return base.DROP
	}

	tf.totalMatched++
	tf.countRetained(record.RawLength)
	return base.PASS
}
