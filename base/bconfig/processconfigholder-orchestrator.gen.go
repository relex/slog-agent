// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package bconfig

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"gopkg.in/yaml.v3"
)

// OrchestratorConfigHolder holds an interface to the actual Config
// The medium is used to support YAML unmarshalling of interfaces
type OrchestratorConfigHolder struct {
	Location string `yaml:"-"`
	OrchestratorConfig
}

// OrchestratorConfigConstructors holds a table of OrchestratorConfig constructors by type name
var OrchestratorConfigConstructors map[string]func() OrchestratorConfig

// RegisterOrchestratorConfigConstructors registers the list of process config structs
// It can only be called once
func RegisterOrchestratorConfigConstructors(newMap map[string]func() OrchestratorConfig) {
	if OrchestratorConfigConstructors != nil {
		logger.Panic("already registered OrchestratorConfigConstructors")
	}
	OrchestratorConfigConstructors = newMap
}

func (holder OrchestratorConfigHolder) String() string {
	return fmt.Sprint(holder.OrchestratorConfig)
}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (holder OrchestratorConfigHolder) MarshalYAML() (interface{}, error) {
	return holder.OrchestratorConfig, nil
}

// UnmarshalYAML provides custom unmarshalling for the implementations of Config
func (holder *OrchestratorConfigHolder) UnmarshalYAML(value *yaml.Node) error {
	if OrchestratorConfigConstructors == nil {
		logger.Panic("OrchestratorConfigConstructors not initialized")
	}
	return unmarshalYAMLObjectHolder(value,
		func(typ string) interface{} {
			createFunc, found := OrchestratorConfigConstructors[typ]
			if found {
				c := createFunc()
				holder.OrchestratorConfig = c
				return c
			}
			return nil
		},
		&holder.Location,
	)
}
