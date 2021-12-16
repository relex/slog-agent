package sysloginput

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/btest"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/input/syslogprotocol"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestSyslogTCPInputConfig(t *testing.T) {
	testMalformedLogLine := "hello world\n"
	testCorrectLogLine := "<163>1 2019-08-15T15:50:46.866915+03:00 local my-app 123 fn - Something\n"
	testOversizedLogLine := "<163>1 2019-08-15T15:50:46.866916+03:00 local my-app 456 fn - Something" + strings.Repeat("x", defs.InputLogMaxMessageBytes) + "\n"

	schema := syslogprotocol.RFC5424Schema
	allocator := base.NewLogAllocator(schema)

	selLevel := schema.MustCreateFieldLocator("level")
	selLog := schema.MustCreateFieldLocator("log")

	config := &Config{}
	if !assert.Nil(t, util.UnmarshalYamlString(`
type: syslog
address: localhost:0
levelMapping: [OFF, FATAL, CRIT, ERROR, WARN, NOTICE, INFO, DEBUG]
extractions:
  - type: delFields
    keys: [facility]
`, config)) {
		return
	}

	stopInput := channels.NewSignalAwaitable()
	logAggregator, outCh := btest.NewLogBufferAggregator(logger.Root())
	mfactory := promreg.NewMetricFactory("test_", nil, nil)

	// create and launch input (the server)
	input, inputErr := config.NewInput(logger.Root(), allocator, schema, logAggregator, mfactory, stopInput)
	if !assert.Nil(t, inputErr) {
		return
	}
	input.Launch()

	// create client connection to send test logs
	conn, cerr := net.Dial("tcp", input.Address())
	assert.Nil(t, cerr)
	_, cerr = conn.Write([]byte(testMalformedLogLine)) // has to be first otherwise it'd be treated as multi-line content
	assert.Nil(t, cerr)
	_, cerr = conn.Write([]byte(testCorrectLogLine))
	assert.Nil(t, cerr)
	_, cerr = conn.Write([]byte(testOversizedLogLine))
	assert.Nil(t, cerr)

	// check resulting logs
	{
		r := readForTest(outCh)
		if assert.Equal(t, 2, len(r)) {
			assert.Equal(t, "ERROR", selLevel.Get(r[0].Fields))
			assert.Equal(t, "Something", selLog.Get(r[0].Fields))
		}
	}

	stopInput.Signal()
	assert.True(t, input.Stopped().Wait(defs.TestReadTimeout))
	assert.Nil(t, conn.Close())

	assert.Equal(t, `test_input_dropped_record_bytes_total{protocol="syslog"} 11
test_input_dropped_records_total{protocol="syslog"} 1
test_input_labelled_record_bytes_total{label="overflow",protocol="syslog"} 1.048647e+06
test_input_labelled_records_total{label="overflow",protocol="syslog"} 1
test_input_passed_record_bytes_total{protocol="syslog"} 1.048718e+06
test_input_passed_records_total{protocol="syslog"} 2
`, promext.DumpMetrics("", true, false, mfactory))
}

func readForTest(ch <-chan []*base.LogRecord) []*base.LogRecord {
	select {
	case logs := <-ch:
		return logs
	case <-time.After(defs.TestReadTimeout):
		return nil
	}
}
