package tparsetime

import (
	"testing"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestParseTimeTransform(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"time"})
	c := &Config{}
	if !assert.Nil(t, util.UnmarshalYamlString(`
type: parseTime
key: time
errorLabel: timeError
`, c)) {
		return
	}
	if !assert.Nil(t, c.VerifyConfig(schema)) {
		return
	}
	tf := c.NewTransform(schema, logger.Root(), bsupport.NewStubLogCustomCounterRegistry())
	{
		record := schema.NewTestRecord2(
			time.Time{},
			base.LogFields{"2019-08-15T15:50:46.866915+03:00"},
		)
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, float64(1565873446.866915000), util.TimeToUnixFloat(record.Timestamp))
	}
	{
		record := schema.NewTestRecord2(
			time.Time{},
			base.LogFields{"2020-09-17T16:51:47.867Z"},
		)
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		tm := record.Timestamp
		assert.Equal(t, time.Month(9), tm.UTC().Month())
		assert.Equal(t, 16, tm.UTC().Hour())
		assert.Equal(t, 51, tm.UTC().Minute())
		assert.Equal(t, 47, tm.UTC().Second())
		assert.Equal(t, 867, tm.UTC().Nanosecond()/1000000)
	}
	{
		record := schema.NewTestRecord2(
			time.Time{},
			base.LogFields{"2020X09-17T16:51:47.867Z"},
		)
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
	}
}
