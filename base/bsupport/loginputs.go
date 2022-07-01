package bsupport

import (
	"fmt"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// VerifyInputConfigs verifies a list of input configurations
func VerifyInputConfigs(inputConfigs []bconfig.LogInputConfigHolder, schema base.LogSchema, header string) error {
	for i, sc := range inputConfigs {
		err := sc.Value.VerifyConfig(schema)
		if err != nil {
			return fmt.Errorf("%s[%d] %s: %w", header, i, sc.Location, err)
		}
	}
	return nil
}
