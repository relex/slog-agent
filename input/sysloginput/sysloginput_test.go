package sysloginput

import (
	"net"
	"testing"
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/input/baseinput"
	"github.com/relex/slog-agent/input/syslogprotocol"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestSyslogTCPInputConfig(t *testing.T) {
	const testLogLine = "<163>1 2019-08-15T15:50:46.866915+03:00 local my-app 123 fn - Something\n"

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
	logAggregator, outCh := baseinput.NewLogBufferAggregator(logger.Root())
	inputMetricFactory := base.NewMetricFactory("test_", nil, nil)

	// create and launch input (the server)
	input, inputErr := config.NewInput(logger.Root(), allocator, schema, logAggregator, inputMetricFactory, stopInput)
	if !assert.Nil(t, inputErr) {
		return
	}
	input.Launch()

	// create client connection to send test logs
	conn, _ := net.Dial("tcp", input.Address())
	_, connErr := conn.Write([]byte(testLogLine))
	assert.Nil(t, connErr)

	// check resulting logs
	{
		r := readForTest(outCh)
		if assert.Equal(t, 1, len(r)) {
			assert.Equal(t, "ERROR", selLevel.Get(r[0].Fields))
			assert.Equal(t, "Something", selLog.Get(r[0].Fields))
		}
	}

	stopInput.Signal()
	assert.True(t, input.Stopped().Wait(defs.TestReadTimeout))
	assert.Nil(t, conn.Close())
	if dump, err := inputMetricFactory.DumpMetrics(true); assert.Nil(t, err) {
		assert.Equal(t, `test_input_dropped_record_bytes_total{protocol="syslog"} 0
test_input_dropped_records_total{protocol="syslog"} 0
test_input_labelled_record_bytes_total{label="overflow",protocol="syslog"} 0
test_input_labelled_records_total{label="overflow",protocol="syslog"} 0
test_input_passed_record_bytes_total{protocol="syslog"} 71
test_input_passed_records_total{protocol="syslog"} 1
`, dump)
	}
}

func readForTest(ch <-chan []*base.LogRecord) []*base.LogRecord {
	select {
	case logs := <-ch:
		return logs
	case <-time.After(defs.TestReadTimeout):
		return nil
	}
}
