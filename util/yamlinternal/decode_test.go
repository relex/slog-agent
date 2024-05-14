package yamlinternal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestDecode(t *testing.T) {
	var node yaml.Node
	assert.NoError(t, yaml.Unmarshal([]byte(`
value: ok

list:
  - "Hi here"
  - "Hey"

unknown: 9
`), &node))

	type ResultType struct {
		Value string   `yaml:"value"`
		List  []string `yaml:"list"`
	}

	assert.Equal(t, "", reflect.TypeOf(node).PkgPath())

	t.Run("decode unknown fields", func(t *testing.T) {
		var result ResultType
		assert.NoError(t, node.Decode(&result))
		assert.Equal(t, "ok", result.Value)
		if assert.Equal(t, 2, len(result.List)) {
			assert.Equal(t, "Hi here", result.List[0])
			assert.Equal(t, "Hey", result.List[1])
		}
	})

	/*
		t.Run("decode unown fields", func(t *testing.T) {
			var result ResultType
			assert.NoError(t, NodeDecodeKnownFields(&node, &result))
			assert.Equal(t, "ok", result.Value)
			if assert.Equal(t, 2, len(result.List)) {
				assert.Equal(t, "Hi here", result.List[0])
				assert.Equal(t, "Hey", result.List[1])
			}
		})
	*/
}
