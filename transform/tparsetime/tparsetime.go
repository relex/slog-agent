// Package tparsetime provides 'parseTime' transform to parses timestamp from a given field.
// Currently only the RFC 3339 timestamp format used in Syslog RFC 5424 is supported.
package tparsetime

import (
	"fmt"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for parseTimeTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Key            string `yaml:"key"`
	ErrorLabel     string `yaml:"errorLabel"`
}

type parseTimeTransform struct {
	keyLocator    base.LogFieldLocator
	timezoneCache map[string]*time.Location
	errorLogger   logger.Logger
	errorCounter  func(length int)
}

// NewTransform creates parseTimeTransform
func (cfg *Config) NewTransform(schema base.LogSchema, parentLogger logger.Logger, customCounterRegistry base.LogCustomCounterRegistry) base.LogTransform {
	tf := &parseTimeTransform{
		keyLocator:    schema.MustCreateFieldLocator(cfg.Key),
		timezoneCache: make(map[string]*time.Location, 100),
		errorLogger:   parentLogger,
		errorCounter:  customCounterRegistry.RegisterCustomCounter(cfg.ErrorLabel),
	}
	return tf
}

// VerifyConfig verifies parseTimeTransform config
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if len(cfg.Key) == 0 {
		return fmt.Errorf(".key is unspecified")
	}
	if _, err := schema.CreateFieldLocator(cfg.Key); err != nil {
		return fmt.Errorf(".key '%s' is invalid: %w", cfg.Key, err)
	}
	if len(cfg.ErrorLabel) == 0 {
		return fmt.Errorf(".errorLabel is unspecified")
	}
	return nil
}

func (tf *parseTimeTransform) Transform(record *base.LogRecord) base.FilterResult {
	value := tf.keyLocator.Get(record.Fields)
	if len(value) == 0 {
		return base.PASS
	}
	tm, err := parseRFC3339Timestamp(value, tf.timezoneCache)
	if err != nil {
		tf.errorCounter(record.RawLength)
		// TODO: omit repeated warnings
		tf.errorLogger.Warnf("failed to parse timestamp: '%s'", value)
	} else {
		record.Timestamp = tm
	}
	return base.PASS
}
