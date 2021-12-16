// Package transform registers the list of all LogTransform implementations
package transform

import (
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/transform/taddfields"
	"github.com/relex/slog-agent/transform/tblock"
	"github.com/relex/slog-agent/transform/tdelfields"
	"github.com/relex/slog-agent/transform/tdrop"
	"github.com/relex/slog-agent/transform/textract"
	"github.com/relex/slog-agent/transform/textractspecial"
	"github.com/relex/slog-agent/transform/tif"
	"github.com/relex/slog-agent/transform/tmapvalue"
	"github.com/relex/slog-agent/transform/tparsetime"
	"github.com/relex/slog-agent/transform/tredactemail"
	"github.com/relex/slog-agent/transform/treplace"
	"github.com/relex/slog-agent/transform/tswitch"
	"github.com/relex/slog-agent/transform/ttruncate"
	"github.com/relex/slog-agent/transform/tunescape"
)

func init() {
	bconfig.RegisterLogTransformConfigConstructors(map[string]func() bconfig.LogTransformConfig{
		"addFields":   func() bconfig.LogTransformConfig { return &taddfields.Config{} },
		"block":       func() bconfig.LogTransformConfig { return &tblock.Config{} },
		"delFields":   func() bconfig.LogTransformConfig { return &tdelfields.Config{} },
		"drop":        func() bconfig.LogTransformConfig { return &tdrop.Config{} },
		"extract":     func() bconfig.LogTransformConfig { return &textract.Config{} },
		"extractHead": func() bconfig.LogTransformConfig { return &textractspecial.Config{} },
		"extractTail": func() bconfig.LogTransformConfig { return &textractspecial.Config{} },
		"if":          func() bconfig.LogTransformConfig { return &tif.Config{} },
		"mapValue":    func() bconfig.LogTransformConfig { return &tmapvalue.Config{} },
		"parseTime":   func() bconfig.LogTransformConfig { return &tparsetime.Config{} },
		"redactEmail": func() bconfig.LogTransformConfig { return &tredactemail.Config{} },
		"replace":     func() bconfig.LogTransformConfig { return &treplace.Config{} },
		"switch":      func() bconfig.LogTransformConfig { return &tswitch.Config{} },
		"truncate":    func() bconfig.LogTransformConfig { return &ttruncate.Config{} },
		"unescape":    func() bconfig.LogTransformConfig { return &tunescape.Config{} },
	})
}

// Register registers all transform config types
func Register() {
	// trigger init()
}
