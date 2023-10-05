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
	{
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
		{
			record := schema.NewTestRecord1(base.LogFields{"HelloWorld"})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "Hello...", record.Fields[0])
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"HelЛWorld"})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "Hel...", record.Fields[0])
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"Foo"})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "Foo", record.Fields[0])
		}
	}
}
