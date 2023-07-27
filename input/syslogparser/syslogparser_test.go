package syslogparser

import (
	"testing"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/input/syslogprotocol"
	"github.com/stretchr/testify/assert"
)

func TestSyslogParser(t *testing.T) {
	schema := syslogprotocol.RFC5424Schema
	allocator := base.NewLogAllocator(schema, 1)
	const line1 = "<163>1 2019-08-15T15:50:46.866915+03:00 local1 my-app1 123 fn1 - Something"
	const line2 = "<163>1 2020-09-17T16:51:47.867Z local2 my-app2 456 fn2 - Something else"
	mfactory := promreg.NewMetricFactory("syslog_parser_", nil, nil)
	counter := base.NewLogInputCounter(mfactory)
	parser, err := NewParser(logger.WithField("test", t.Name()), allocator, schema, syslogprotocol.SeverityNames, counter)
	assert.NoError(t, err)
	{
		tm := time.Now()
		r1 := parser.Parse([]byte(line1), tm)
		if assert.NotNil(t, r1) {
			assert.Equal(t, "my-app1", schema.MustCreateFieldLocator("app").Get(r1.Fields))
			assert.Equal(t, "123", schema.MustCreateFieldLocator("pid").Get(r1.Fields))
			assert.Equal(t, "fn1", schema.MustCreateFieldLocator("source").Get(r1.Fields))
			assert.Equal(t, tm, r1.Timestamp)
		}
	}
	{
		r2 := parser.Parse([]byte(line2), time.Now())
		if assert.NotNil(t, r2) {
			assert.Equal(t, "my-app2", schema.MustCreateFieldLocator("app").Get(r2.Fields))
			assert.Equal(t, "456", schema.MustCreateFieldLocator("pid").Get(r2.Fields))
			assert.Equal(t, "local4", schema.MustCreateFieldLocator("facility").Get(r2.Fields))
			assert.Equal(t, "err", schema.MustCreateFieldLocator("level").Get(r2.Fields))
			assert.Equal(t, "my-app2", schema.MustCreateFieldLocator("app").Get(r2.Fields))
			assert.Equal(t, "456", schema.MustCreateFieldLocator("pid").Get(r2.Fields))
		}
	}
	counter.UpdateMetrics()

	assert.Equal(t, `syslog_parser_dropped_record_bytes_total 0
syslog_parser_dropped_records_total 0
syslog_parser_passed_record_bytes_total 145
syslog_parser_passed_records_total 2
`, promext.DumpMetrics("", true, false, mfactory))
}
