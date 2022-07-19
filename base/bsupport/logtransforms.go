package bsupport

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/defs"
)

// RunTransforms executes all the given transforms on a log record
func RunTransforms(record *base.LogRecord, transforms []base.LogTransformFunc) base.FilterResult {
	for _, transformFunc := range transforms {
		if transformFunc(record) == base.DROP {
			return base.DROP
		}
	}
	return base.PASS
}

// NewTransformsFromConfig creates transforms from a list of transform configurations
func NewTransformsFromConfig(transformConfigs []bconfig.LogTransformConfigHolder, schema base.LogSchema,
	parentLogger logger.Logger, customCounterHost base.LogCustomCounterRegistry,
) []base.LogTransformFunc {
	transforms := make([]base.LogTransformFunc, len(transformConfigs))
	for i, tc := range transformConfigs {
		tlogger := parentLogger.WithFields(logger.Fields{
			defs.LabelPart:   tc.Value.GetType(),
			defs.LabelSource: tc.Location,
		})
		transforms[i] = tc.Value.NewTransform(schema, tlogger, customCounterHost).Transform
	}
	return transforms
}

// VerifyTransformConfigs verifies a list of transform configurations
func VerifyTransformConfigs(transformConfigs []bconfig.LogTransformConfigHolder, schema base.LogSchema, header string) error {
	for i, tfc := range transformConfigs {
		err := tfc.Value.VerifyConfig(schema)
		if err != nil {
			return fmt.Errorf("%s[%d] %s: %w", header, i, tfc.Location, err)
		}
	}
	return nil
}
