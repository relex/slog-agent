package bconfig

import (
	"fmt"
	"reflect"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/util"
	"github.com/relex/slog-agent/util/yamlinternal"
	"gopkg.in/yaml.v3"
)

// ConfigHolder holds an interface to the actual Config
//
// The medium is used to support YAML unmarshalling of interfaces
type ConfigHolder[C BaseConfig] struct {
	Location string `yaml:"-"`
	Value    C
}

func (holder ConfigHolder[C]) String() string {
	return fmt.Sprint(holder.Value)
}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (holder ConfigHolder[C]) MarshalYAML() (interface{}, error) {
	return holder.Value, nil
}

// UnmarshalYAML provides custom unmarshalling for the implementations of Config
func (holder *ConfigHolder[C]) UnmarshalYAML(value *yaml.Node) error {
	table := getConfigConstructors[C]()

	// Find type
	if len(value.Content) < 2 {
		return util.NewYamlError(value, ".type is undefined")
	}
	if value.Content[0].Kind != yaml.ScalarNode || value.Content[0].Value != "type" {
		return util.NewYamlError(value, fmt.Sprintf(".type is not the first property, which is: %v", value.Content[0]))
	}
	typeName := value.Content[1].Value

	createFunc, found := table[typeName]
	if !found {
		return util.NewYamlError(value, fmt.Sprintf(".type: unsupported '%s'", typeName))
	}
	holder.Value = createFunc()

	if err := yamlinternal.NodeDecodeKnownFields(value, holder.Value); err != nil {
		return util.NewYamlError(value, err.Error())
	}
	holder.Location = util.GetYamlLocation(value)
	return nil
}

// ConfigCreatorTable provides a map of config types to their constructors
type ConfigCreatorTable[C BaseConfig] map[string]func() C

var typeToConfigCreatorTables = make(map[string]interface{})

// RegisterConfigConstructors registers the list of config constructors for a particular config type
//
// It can only be called once for each type C
func RegisterConfigConstructors[C BaseConfig](newMap ConfigCreatorTable[C]) {
	c := reflect.TypeOf((*C)(nil)).Elem()
	_, exists := typeToConfigCreatorTables[c.String()]
	if exists {
		logger.Panicf("already registered %s", c.String())
	}
	typeToConfigCreatorTables[c.String()] = newMap
}

func getConfigConstructors[C BaseConfig]() ConfigCreatorTable[C] {
	c := reflect.TypeOf((*C)(nil)).Elem()
	table, exists := typeToConfigCreatorTables[c.String()]
	if !exists {
		logger.Panicf("not registered %s", c.String())
	}
	return table.(ConfigCreatorTable[C])
}
