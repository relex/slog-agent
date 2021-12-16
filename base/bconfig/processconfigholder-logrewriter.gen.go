// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package bconfig

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"gopkg.in/yaml.v3"
)

// LogRewriterConfigHolder holds an interface to the actual Config
//
// The medium is used to support YAML unmarshalling of interfaces
type LogRewriterConfigHolder struct {
	Location string `yaml:"-"`
	LogRewriterConfig
}

// LogRewriterConfigConstructors holds a table of LogRewriterConfig constructors by type name
var LogRewriterConfigConstructors map[string]func() LogRewriterConfig

// RegisterLogRewriterConfigConstructors registers the list of process config structs
//
// It can only be called once
func RegisterLogRewriterConfigConstructors(newMap map[string]func() LogRewriterConfig) {
	if LogRewriterConfigConstructors != nil {
		logger.Panic("already registered LogRewriterConfigConstructors")
	}
	LogRewriterConfigConstructors = newMap
}

func (holder LogRewriterConfigHolder) String() string {
	return fmt.Sprint(holder.LogRewriterConfig)
}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (holder LogRewriterConfigHolder) MarshalYAML() (interface{}, error) {
	return holder.LogRewriterConfig, nil
}

// UnmarshalYAML provides custom unmarshalling for the implementations of Config
func (holder *LogRewriterConfigHolder) UnmarshalYAML(value *yaml.Node) error {
	if LogRewriterConfigConstructors == nil {
		logger.Panic("LogRewriterConfigConstructors not initialized")
	}
	return unmarshalYAMLObjectHolder(value,
		func(typ string) interface{} {
			createFunc, found := LogRewriterConfigConstructors[typ]
			if found {
				c := createFunc()
				holder.LogRewriterConfig = c
				return c
			}
			return nil
		},
		&holder.Location,
	)
}
