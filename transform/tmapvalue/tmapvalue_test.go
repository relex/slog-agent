package tmapvalue

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestMapValueTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"severity"})
	c := &Config{}
	assert.Nil(t, util.UnmarshalYamlString(`
type: mapValue
key: severity
mapping:
  emergency: FATAL
  warning: WARN
default: UNKNOWN
`, c))
	tf := c.NewTransform(schema, logger.Root(), nil)
	{
		record := schema.NewTestRecord1(base.LogFields{"emergency"})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, "FATAL", record.Fields[0])
	}
	{
		record := schema.NewTestRecord1(base.LogFields{"tracing"})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, "UNKNOWN", record.Fields[0])
	}
}
