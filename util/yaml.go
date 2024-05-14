package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// GetYamlLocation fetches a descriptive location of YAML node
func GetYamlLocation(node *yaml.Node) string {
	var title string
	switch {
	case len(node.HeadComment) > 0:
		title = " " + node.HeadComment
	case len(node.Anchor) > 0:
		title = " " + node.Anchor
	default:
		title = ""
	}
	return fmt.Sprintf("yaml line %d:%d%s", node.Line, node.Column, title)
}

// MarshalYaml marshals the given source to a YAML string
func MarshalYaml(source interface{}) (string, error) {
	writer := &bytes.Buffer{}
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	if err := encoder.Encode(source); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return writer.String(), nil
}

// NewYamlError creates a new error with location information of YAML node
func NewYamlError(node *yaml.Node, message string) error {
	return fmt.Errorf("yaml line %d:%d: %s", node.Line, node.Column, message)
}

// UnmarshalYamlFile loads and unmarshals YAML from file to interface or pointer to struct
func UnmarshalYamlFile(path string, output interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return UnmarshalYamlReader(file, output)
}

// UnmarshalYamlReader loads and unmarshals YAML from IO reader to interface or pointer to struct
func UnmarshalYamlReader(reader io.Reader, output interface{}) error {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true) // only works outside of custom unmarshalers
	return decoder.Decode(output)
}

// UnmarshalYamlString loads and unmarshals YAML in string to interface or pointer to struct
func UnmarshalYamlString(contents string, output interface{}) error {
	reader := strings.NewReader(contents)
	return UnmarshalYamlReader(reader, output)
}
