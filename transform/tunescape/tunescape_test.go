package tunescape

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestUnescapeTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"msg"})
	c := &Config{}
	assert.NoError(t, util.UnmarshalYamlString("type: unescape\nkey: msg", c))
	assert.NoError(t, c.VerifyConfig(schema))
	tf := c.NewTransform(schema, logger.Root(), nil)
	{
		record := schema.NewTestRecord1(base.LogFields{`x\Xhello\n`})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, "x\\Xhello\n", record.Fields[0])
	}
	{
		record := schema.NewTestRecord1(base.LogFields{"unchanged"})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, "unchanged", record.Fields[0])
	}
}

func TestUnescapeTransformConfig(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"msg", "msg2"})
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString("type: unescape\nkey: msg2", c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		record := schema.NewTestRecord1(base.LogFields{`x\nHello`, `x\nWorld`})
		_ = tf.Transform(record)
		assert.Equal(t, "x\\nHello", record.Fields[0])
		assert.Equal(t, "x\nWorld", record.Fields[1])
	}
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString("type: unescape\nkey: ''", c))
		assert.EqualError(t, c.VerifyConfig(schema), ".key is unspecified")
	}
}
