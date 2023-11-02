package ttruncate

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestTruncateTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"message"})
	c := &Config{}
	if !assert.NoError(t, util.UnmarshalYamlString(`
type: truncate
key: message
maxLen: 5
suffix: ...
`, c)) {
		return
	}
	if !assert.NoError(t, c.VerifyConfig(schema)) {
		return
	}
	tf := c.NewTransform(schema, logger.Root(), nil)

	testCases := []struct {
		Input    string
		Expected string
	}{
		{"", ""},
		{"Foo", "Foo"},
		{"Hello123", "Hello123"}, // no change since len = max + suffix
		{"HelloWorld", "Hello..."},
		{"1234ЛWorld", "1234..."}, // cut incomplete utf-8
		{"1234世界World", "1234..."},
		{"123世界World", "123..."},
		{"12世界World", "12世..."},
	}

	for i, test := range testCases {
		record := schema.NewTestRecord1(base.LogFields{util.StringFromBytes([]byte(test.Input))})
		status := tf.Transform(record)
		assert.Equalf(t, base.PASS, status, "transformed[%d] %s", i, test.Input)
		assert.Equalf(t, test.Expected, record.Fields[0], "resulted[%d] %s", i, test.Input)
	}
}
