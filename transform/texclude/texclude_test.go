package texclude

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestExcludeTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"msg"})
	c := &Config{}
	assert.Nil(t, util.UnmarshalYamlString(`
type: exclude
match:
  msg: !!str-start Hi
percentage: 60
exclusionLabel: HiDrop
retentionLabel: HiPass
`, c))
	reg := bsupport.NewStubLogCustomCounterRegistry()
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
}
