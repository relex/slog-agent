package textractspecial

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestExtractHeadTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"filename", "id"})
	iFilename := 0
	iID := 1
	{
		c := &Config{}
		if !assert.Nil(t, util.UnmarshalYamlString(`
type: extractHead
key: filename
pattern: '[a-z0-9].'
maxLen: 10
destKey: id
`, c)) {
			return
		}
		if !assert.Nil(t, c.VerifyConfig(schema)) {
			return
		}
		tf := c.NewTransform(schema, logger.Root(), nil)
		{
			record := schema.NewTestRecord1(base.LogFields{"a12345.txt", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "txt", record.Fields[iFilename])
			assert.Equal(t, "a12345", record.Fields[iID])
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"0123456789.txt", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "0123456789.txt", record.Fields[iFilename])
			assert.Equal(t, "", record.Fields[iID])
		}
	}
}

func TestExtractTailTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"filename", "num"})
	iFilename := 0
	iID := 1
	{
		c := &Config{}
		assert.Nil(t, util.UnmarshalYamlString(`
type: extractTail
key: filename
pattern: '.[0-9]$'
maxLen: 10
destKey: num
`, c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		{
			record := schema.NewTestRecord1(base.LogFields{"hello.123$", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "hello", record.Fields[iFilename])
			assert.Equal(t, "123", record.Fields[iID])
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"foo.12x$", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "foo.12x$", record.Fields[iFilename])
			assert.Equal(t, "", record.Fields[iID])
		}
	}
}
