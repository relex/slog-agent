package tdrop

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/btest"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestDropTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"name", "alt"})
	c := &Config{}
	assert.NoError(t, util.UnmarshalYamlString(`
type: drop
match:
  name: Foo
percentage: 100
metricLabel: test
`, c))
	reg, lookup := btest.NewStubLogCustomCounterRegistry()
	tf := c.NewTransform(schema, logger.Root(), reg)
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
	{
		cnt, len := lookup("test")
		assert.Equal(t, int64(2), cnt)
		assert.Equal(t, int64(21), len)
	}
}

func TestDropTransformWithPercentage(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"msg"})
	c := &Config{}
	assert.NoError(t, util.UnmarshalYamlString(`
type: drop
match:
  msg: !!str-start Hi
percentage: 60
metricLabel: HiDropped
`, c))
	reg, lookup := btest.NewStubLogCustomCounterRegistry()
	tf := c.NewTransform(schema, logger.Root(), reg)
	tfRec := func(msg string) base.FilterResult {
		record := schema.NewTestRecord1(base.LogFields{msg})
		record.RawLength = 10
		return tf.Transform(record)
	}
	assert.Equal(t, base.PASS, tfRec("Hi 1"))
	assert.Equal(t, base.DROP, tfRec("Hi 2"))
	assert.Equal(t, base.DROP, tfRec("Hi 3"))
	assert.Equal(t, base.PASS, tfRec("Hi 4")) // pass because drop test is current rate < target rate
	assert.Equal(t, base.DROP, tfRec("Hi 5"))
	assert.Equal(t, base.PASS, tfRec("Hi 6"))
	assert.Equal(t, base.DROP, tfRec("Hi 7"))
	assert.Equal(t, base.DROP, tfRec("Hi 8"))
	assert.Equal(t, base.PASS, tfRec("Hi 9"))
	assert.Equal(t, base.DROP, tfRec("Hi 10"))
	{
		cnt, _ := lookup("HiDropped")
		assert.Equal(t, int64(6), cnt)
	}
	{
		cnt, _ := lookup("!HiDropped")
		assert.Equal(t, int64(4), cnt)
	}
}
