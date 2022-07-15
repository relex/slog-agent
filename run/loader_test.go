package run

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/relex/fluentlib/server"
	"github.com/relex/fluentlib/server/receivers"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/stretchr/testify/assert"
)

const sampleSchemaConf = `
schema:
  fields: [facility, level, time, host, app, pid, source, extradata, log, sn]
  maxFields: 12
`

const sampleInputConf = `
inputs:
  - type: syslog
    address: localhost:0
    levelMapping: [off, fatal, crit, error, warn, notice, info, debug]
    extractions:
      - type: delFields
        keys: [facility, pid]
`

const sampleOrchestrationConf = `
orchestration:
  type: byKeySet
  keys: [app]
  tag: development.$app
metricKeys: [host]
`

const sampleTransformationConf = `
transformations:
  - type: extractTail
    key: log
    pattern: ', [0-9]'
    maxLen: 20
    destKey: sn
  - type: addFields
    fields:
      log: $log hello
  - type: parseTime
    key: time
    errorLabel: timeError
  - type: redactEmail
    key: log
    metricLabel: emailFilter
`

const sampleOutputConf = `
outputBufferPairs:
  - name: testPairName
    buffer:
        type: hybridBuffer
        rootPath: %s
        maxBufSize: 1GB
    output:
        type: fluentdForward
        serialization:
            environmentFields: [host, app, source]
            hiddenFields: []
            rewriteFields:
                log:
                    - type: unescape
        messageMode: CompressedPackedForward
        upstream:
            address: %s
            tls: false
            maxDuration: 500ms
`

var sampleConf = assembleConfig(
	sampleSchemaConf,
	sampleInputConf,
	sampleOrchestrationConf,
	sampleTransformationConf,
	sampleOutputConf,
)

func TestLoader(t *testing.T) {
	logRecv, outBatchCh := receivers.NewMessageCollector(5 * time.Second)

	runTestEnv(t, logRecv, sampleConf, func(bufDir string, confFile *os.File, srvAddr net.Addr) {
		ld, confErr := NewLoaderFromConfigFile(confFile.Name(), t.Name()+"_")
		assert.Nil(t, confErr)
		assert.Equal(t, []string{"facility", "level", "time", "host", "app", "pid", "source", "extradata", "log"}, ld.ConfigStats.FixedFields)
		assert.Empty(t, ld.ConfigStats.UnusedFields)

		orc := ld.StartOrchestrator(logger.Root())

		inputAddrs, shutdownInputs := ld.LaunchInputs(orc)
		assert.Equal(t, 1, len(inputAddrs))

		metricGatherer := ld.GetMetricGatherer()

		conn, connErr := net.Dial("tcp", inputAddrs[0])
		assert.Nil(t, connErr)

		// send logs
		// DO NOT use testdata here - need to have different config(s) and different outputs
		_, sendErr := conn.Write([]byte(`<167>1 2020-07-20T03:48:20.154+03:00 host1 appServ/foo.com 51629 cron.log - [Initializer] - Hello Foo
<166>1 2020-07-20T03:48:33.760+03:00 host1 appServ/foo.com 51629 access.log - Hello Bar me@gmail.com
`))
		assert.Nil(t, sendErr)

		// check server results
		result := <-outBatchCh
		assert.Equal(t, 2, len(result.Entries))
		assert.Equal(t, "[Initializer] - Hello Foo hello", result.Entries[0].Record["log"])
		assert.Equal(t, "info", result.Entries[1].Record["level"])
		assert.Nil(t, conn.Close())

		shutdownInputs()
		orc.Shutdown()

		metricFamilies, promErr := metricGatherer.Gather()
		assert.Nil(t, promErr)
		assert.Equal(t, float64(1), getMetricValue(t, metricFamilies, "process_labelled_records_total",
			map[string]string{"label": "emailFilter"}))
		assert.Equal(t, float64(2), getMetricValue(t, metricFamilies, "input_passed_records_total",
			map[string]string{"protocol": "syslog"}))
	})
}

func assembleConfig(parts ...string) string {
	return `
anchors: []
` + strings.Join(parts, "")
}

func runTestEnv(t *testing.T, logReceiver receivers.Receiver, confYML string,
	do func(bufDir string, confFile *os.File, srvAddr net.Addr)) {

	bufDir, bufDirErr := os.MkdirTemp("", fmt.Sprintf("slog-agent-%s-buf-*", t.Name()))
	assert.Nil(t, bufDirErr)
	defer os.RemoveAll(bufDir)

	confFile, confFileErr := os.CreateTemp("", fmt.Sprintf("slog-agent-%s-conf-*.yml", t.Name()))
	assert.Nil(t, confFileErr)
	defer os.Remove(confFile.Name())

	srvConf := server.Config{}
	srvConf.Address = "localhost:0"
	srv, srvAddr := server.LaunchServer(logger.WithField("test", t.Name()), srvConf, logReceiver)

	_, writeErr := confFile.WriteString(fmt.Sprintf(confYML, bufDir, srvAddr.String()))
	assert.Nil(t, writeErr)
	assert.Nil(t, confFile.Close())

	defer srv.Shutdown()

	do(bufDir, confFile, srvAddr)
}

func getMetricValue(t *testing.T, metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	mf := findMetricFamily(t, metricFamilies, name)
	if !assert.NotNil(t, mf, "find ", t.Name()+"_"+name) {
		return 0
	}

	return promext.SumExportedMetrics(mf, labels)
}

func findMetricFamily(t *testing.T, metricFamilies []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, mf := range metricFamilies {
		if *mf.Name == t.Name()+"_"+name {
			return mf
		}
	}
	return nil
}
