// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package bconfig

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// LogOutputConfigHolder holds an interface to the actual Config
// The medium is used to support YAML unmarshalling of interfaces
type LogOutputConfigHolder struct {
	Location string `yaml:"-"`
	LogOutputConfig
}

// LogOutputConfigConstructors holds a table of LogOutputConfig constructors by type name
var LogOutputConfigConstructors map[string]func() LogOutputConfig

// RegisterLogOutputConfigConstructors registers the list of process config structs
// It can only be called once
func RegisterLogOutputConfigConstructors(newMap map[string]func() LogOutputConfig) {
	if LogOutputConfigConstructors != nil {
		panic("already registered LogOutputConfigConstructors")
	}
	LogOutputConfigConstructors = newMap
}

func (holder LogOutputConfigHolder) String() string {
	return fmt.Sprint(holder.LogOutputConfig)
}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (holder LogOutputConfigHolder) MarshalYAML() (interface{}, error) {
	return holder.LogOutputConfig, nil
}

// UnmarshalYAML provides custom unmarshalling for the implementations of Config
func (holder *LogOutputConfigHolder) UnmarshalYAML(value *yaml.Node) error {
	if LogOutputConfigConstructors == nil {
		panic("LogOutputConfigConstructors not initialized")
	}
	return unmarshalYAMLObjectHolder(value,
		func(typ string) interface{} {
			createFunc, found := LogOutputConfigConstructors[typ]
			if found {
				c := createFunc()
				holder.LogOutputConfig = c
				return c
			}
			return nil
		},
		&holder.Location,
	)
}
