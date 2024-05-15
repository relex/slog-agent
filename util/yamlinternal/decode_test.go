package yamlinternal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNodeDecodeKnownFields(t *testing.T) {
	var correctNode yaml.Node
	correctYaml := `
value: ok

list:
  - "Hi here"
  - "Hey correct"
`

	var incorrectNode yaml.Node
	incorrectYaml := `
value: ok

list:
  - "Hi here"
  - "Hey incorrect"

unknown: 9
`
	assert.NoError(t, yaml.Unmarshal([]byte(correctYaml), &correctNode))
	assert.NoError(t, yaml.Unmarshal([]byte(incorrectYaml), &incorrectNode))

	type ResultType struct {
		Value string   `yaml:"value"`
		List  []string `yaml:"list"`
	}

	t.Run("decode with builtin method to accept unknown fields", func(t *testing.T) {
		var result ResultType
		assert.NoError(t, incorrectNode.Decode(&result))
		assert.Equal(t, "ok", result.Value)
		if assert.Equal(t, 2, len(result.List)) {
			assert.Equal(t, "Hi here", result.List[0])
			assert.Equal(t, "Hey incorrect", result.List[1])
		}
	})

	t.Run("decode with new method to reject unknown fields", func(t *testing.T) {
		var result ResultType
		assert.ErrorContains(t, NodeDecodeKnownFields(&incorrectNode, &result), "line 8: field unknown not found in type yamlinternal.ResultType")
	})

	t.Run("decode with new method to accept known fields", func(t *testing.T) {
		var result ResultType
		assert.NoError(t, NodeDecodeKnownFields(&correctNode, &result))
		assert.Equal(t, "ok", result.Value)
		if assert.Equal(t, 2, len(result.List)) {
			assert.Equal(t, "Hi here", result.List[0])
			assert.Equal(t, "Hey correct", result.List[1])
		}
	})
}
