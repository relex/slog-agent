package bconfig

import (
	"fmt"

	"github.com/relex/slog-agent/util"
	"gopkg.in/yaml.v3"
)

// unmarshalYAMLObjectHolder is a helper method to unmarshal type-specific configuration of element to a holder field (ex: LogInput or LogTransform)
func unmarshalYAMLObjectHolder(value *yaml.Node, createConfig func(typ string) interface{}, locationPtr *string) error {
	// Find type
	if len(value.Content) < 2 {
		return util.NewYamlError(value, ".type is undefined")
	}
	if value.Content[0].Kind != yaml.ScalarNode || value.Content[0].Value != "type" {
		return util.NewYamlError(value, fmt.Sprintf(".type is not the first property, which is: %v", value.Content[0]))
	}
	typeName := value.Content[1].Value
	// Create the real config
	config := createConfig(typeName)
	if config == nil {
		return util.NewYamlError(value, fmt.Sprintf(".type: unsupported '%s'", typeName))
	}
	if err := value.Decode(config); err != nil {
		return util.NewYamlError(value, err.Error())
	}
	if locationPtr != nil {
		*locationPtr = util.GetYamlLocation(value)
	}
	return nil
}
