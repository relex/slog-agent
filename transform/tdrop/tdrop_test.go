package tdrop

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestDropTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"name", "alt"})
	{
		c := &Config{}
		assert.Nil(t, util.UnmarshalYamlString(`
type: drop
match:
  name: Foo
metricLabel: test
`, c))
		tf := c.NewTransform(schema, logger.Root(), bsupport.NewStubLogCustomCounterRegistry())
		{
			record := schema.NewTestRecord1(base.LogFields{"Foo", ""})
			record.RawLength = 10
			status := tf.Transform(record)
			assert.Equal(t, base.DROP, status)
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"Bar", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, 3, len(record.Fields[0]))
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"Foo", "2"})
			record.RawLength = 11
			status := tf.Transform(record)
			assert.Equal(t, base.DROP, status)
		}
	}
}
