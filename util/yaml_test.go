package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type yamlParentType struct {
	Name  string
	Child yamlChildType
}

type yamlChildType string

var yamlTestTempLocation string

func (yc *yamlChildType) UnmarshalYAML(node *yaml.Node) error {
	yamlTestTempLocation = GetYamlLocation(node)
	if node.Value == "fail" {
		return NewYamlError(node, "Fail")
	}
	*yc = yamlChildType(node.Value)
	return nil
}

func TestYAMLMarshal(t *testing.T) {
	y, err := MarshalYaml(&yamlParentType{
		Name:  "succ",
		Child: yamlChildType("here"),
	})
	assert.Nil(t, err)
	assert.Equal(t, "name: succ\nchild: here\n", y)
}

func TestYAMLUnmarshal(t *testing.T) {
	var yp yamlParentType

	assert.ErrorContains(t, UnmarshalYamlString(`
name: hi
child: fail
`, &yp), "yaml line 3:8: Fail")
	assert.Equal(t, "yaml line 3:8", yamlTestTempLocation)
}
