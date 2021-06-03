package bconfig

//go:generate genny -in=$GOFILE -out=processconfigholder-chunkbuffer.gen.go gen "ProcessConfig=ChunkBufferConfig Process=ChunkBuffer"
//go:generate genny -in=$GOFILE -out=processconfigholder-loginput.gen.go gen "ProcessConfig=LogInputConfig Process=LogInput"
//go:generate genny -in=$GOFILE -out=processconfigholder-logoutput.gen.go gen "ProcessConfig=LogOutputConfig Process=LogOutput"
//go:generate genny -in=$GOFILE -out=processconfigholder-logrewriter.gen.go gen "ProcessConfig=LogRewriterConfig Process=LogRewriter"
//go:generate genny -in=$GOFILE -out=processconfigholder-logtransform.gen.go gen "ProcessConfig=LogTransformConfig Process=LogTransform"
//go:generate genny -in=$GOFILE -out=processconfigholder-orchestrator.gen.go gen "ProcessConfig=OrchestratorConfig Process=Orchestrator"

import (
	"fmt"

	"github.com/cheekybits/genny/generic"
	"gopkg.in/yaml.v3"
)

// ProcessConfig is the type of item in the process channel to PipelineWorkerBase
type ProcessConfig generic.Type

// ProcessConfigHolder holds an interface to the actual Config
// The medium is used to support YAML unmarshalling of interfaces
type ProcessConfigHolder struct {
	Location string `yaml:"-"`
	ProcessConfig
}

// ProcessConfigConstructors holds a table of ProcessConfig constructors by type name
var ProcessConfigConstructors map[string]func() ProcessConfig

// RegisterProcessConfigConstructors registers the list of process config structs
// It can only be called once
func RegisterProcessConfigConstructors(newMap map[string]func() ProcessConfig) {
	if ProcessConfigConstructors != nil {
		panic("already registered ProcessConfigConstructors")
	}
	ProcessConfigConstructors = newMap
}

func (holder ProcessConfigHolder) String() string {
	return fmt.Sprint(holder.ProcessConfig)
}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (holder ProcessConfigHolder) MarshalYAML() (interface{}, error) {
	return holder.ProcessConfig, nil
}

// UnmarshalYAML provides custom unmarshalling for the implementations of Config
func (holder *ProcessConfigHolder) UnmarshalYAML(value *yaml.Node) error {
	if ProcessConfigConstructors == nil {
		panic("ProcessConfigConstructors not initialized")
	}
	return unmarshalYAMLObjectHolder(value,
		func(typ string) interface{} {
			createFunc, found := ProcessConfigConstructors[typ]
			if found {
				c := createFunc()
				holder.ProcessConfig = c
				return c
			}
			return nil
		},
		&holder.Location,
	)
}
