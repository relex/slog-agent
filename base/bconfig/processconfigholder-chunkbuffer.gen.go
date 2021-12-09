// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package bconfig

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"gopkg.in/yaml.v3"
)

// ChunkBufferConfigHolder holds an interface to the actual Config
//
// The medium is used to support YAML unmarshalling of interfaces
type ChunkBufferConfigHolder struct {
	Location string `yaml:"-"`
	ChunkBufferConfig
}

// ChunkBufferConfigConstructors holds a table of ChunkBufferConfig constructors by type name
var ChunkBufferConfigConstructors map[string]func() ChunkBufferConfig

// RegisterChunkBufferConfigConstructors registers the list of process config structs
//
// It can only be called once
func RegisterChunkBufferConfigConstructors(newMap map[string]func() ChunkBufferConfig) {
	if ChunkBufferConfigConstructors != nil {
		logger.Panic("already registered ChunkBufferConfigConstructors")
	}
	ChunkBufferConfigConstructors = newMap
}

func (holder ChunkBufferConfigHolder) String() string {
	return fmt.Sprint(holder.ChunkBufferConfig)
}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (holder ChunkBufferConfigHolder) MarshalYAML() (interface{}, error) {
	return holder.ChunkBufferConfig, nil
}

// UnmarshalYAML provides custom unmarshalling for the implementations of Config
func (holder *ChunkBufferConfigHolder) UnmarshalYAML(value *yaml.Node) error {
	if ChunkBufferConfigConstructors == nil {
		logger.Panic("ChunkBufferConfigConstructors not initialized")
	}
	return unmarshalYAMLObjectHolder(value,
		func(typ string) interface{} {
			createFunc, found := ChunkBufferConfigConstructors[typ]
			if found {
				c := createFunc()
				holder.ChunkBufferConfig = c
				return c
			}
			return nil
		},
		&holder.Location,
	)
}
