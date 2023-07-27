package textract

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestExtractTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"msgid", "folder", "source", "timestamp"})
	iFolder := 1
	iSource := 2
	iTimestamp := 3
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString(`
type: extract
key: msgid
pattern: ^((?P<folder>[^ /]+)/)?(?P<source>[^ ]*?)(\.(?P<timestamp>-?[0-9]+))?$
`, c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		{
			record := schema.NewTestRecord1(base.LogFields{"production/hourly.log.202012311030", "", "", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "production", record.Fields[iFolder])
			assert.Equal(t, "hourly.log", record.Fields[iSource])
			assert.Equal(t, "202012311030", record.Fields[iTimestamp])
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"testing/main.log", "", "", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "testing", record.Fields[iFolder])
			assert.Equal(t, "main.log", record.Fields[iSource])
			assert.Equal(t, "", record.Fields[iTimestamp])
		}
	}
}
