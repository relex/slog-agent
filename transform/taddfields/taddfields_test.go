package taddfields

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestAddFieldsTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"ident", "pid", "message", "type", "app", "log"})
	iType := 3
	iApp := 4
	iLog := 5
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString(`
type: addFields
fields:
  app: $ident($pid)
  log: $message
  type: development
`, c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		record := schema.NewTestRecord1(base.LogFields{"notepad", "11", "open foo.bar", "undefined", "", ""})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, "notepad(11)", record.Fields[iApp])
		assert.Equal(t, "open foo.bar", record.Fields[iLog])
		assert.Equal(t, "development", record.Fields[iType])
	}
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString(`
type: addFields
fields:
  app: $ident($pid)
  log: $message${x
`, c))
		assert.EqualError(t, c.VerifyConfig(schema), ".fields[log] has invalid template: unenclosed variable quotes: '$message${x'")

	}
}
