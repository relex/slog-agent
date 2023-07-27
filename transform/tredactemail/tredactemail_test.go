package tredactemail

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/btest"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestRedactEmailTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"log"})
	c := &Config{}
	assert.NoError(t, util.UnmarshalYamlString(`
type: redactEmail
key: log
metricLabel: email
`, c))
	reg, _ := btest.NewStubLogCustomCounterRegistry()
	tf := c.NewTransform(schema, logger.Root(), reg)
	{
		record := schema.NewTestRecord1(base.LogFields{"reply_to: foo@bar.com, john@x.com something@else.org,"})
		_ = tf.Transform(record)
		assert.Equal(t, "reply_to: REDACTED, REDACTED REDACTED,", record.Fields[0])
	}
}

func TestRedactEmailTransformVerify(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"log"})
	c := &Config{}
	assert.NoError(t, util.UnmarshalYamlString("type: redactEmail\nkey: ''", c))
	assert.EqualError(t, c.VerifyConfig(schema), ".key is unspecified")
}
