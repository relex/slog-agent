package treplace

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestReplaceTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"log"})
	{
		c := &Config{}
		assert.Nil(t, util.UnmarshalYamlString(`
type: replace
key: log
pattern: '([Ee]-?mail[:=]*) *"?[\w.-]+@[\w.-]+"?'
replacement: '$1 "REDACTED"'
`, c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		record := schema.NewTestRecord1(base.LogFields{`Email: foo@not-real-gmail.com, e-mail: hey-bar@yahoo.com`})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, `Email: "REDACTED", e-mail: "REDACTED"`, record.Fields[0])
	}
}
