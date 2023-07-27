package tdelfields

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestDelFieldsTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"a", "b", "c"})
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString(`
type: delFields
keys: [a, b]
`, c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		record := schema.NewTestRecord1(base.LogFields{"foo", "bar", "xxx"})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, "", schema.MustCreateFieldLocator("a").Get(record.Fields))
		assert.Equal(t, "", schema.MustCreateFieldLocator("b").Get(record.Fields))
		assert.Equal(t, "xxx", schema.MustCreateFieldLocator("c").Get(record.Fields))
	}
}
