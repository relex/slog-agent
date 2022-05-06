package run

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/fluentlib/server/receivers"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/stretchr/testify/assert"
)

var sampleYML2 = assembleConfig(
	strings.ReplaceAll(sampleSchemaConf, ", log, sn]", ", log, sn2, sn]"),
	sampleInputConf,
	sampleOrchestrationConf,
	strings.ReplaceAll(strings.ReplaceAll(sampleTransformationConf,
		"log: $log hello", "log: $log new"),
		"emailFilter", "MyMailFilter")+`
  - type: addFields
    fields:
      sn2: New $sn
`,
	sampleOutputConf,
)

func TestReloader(t *testing.T) {
	logRecv, outBatchCh := receivers.NewEventCollector(5 * time.Second)

	runTestEnv(t, logRecv, sampleConf, func(bufDir string, confFile *os.File, srvAddr net.Addr) {
		ld, confErr := NewReloaderFromConfigFile(confFile.Name(), t.Name()+"_")
		assert.Nil(t, confErr)
		assert.Empty(t, ld.ConfigStats.UnusedFields)

		orc := ld.LaunchOrchestrator(logger.Root())

		inAddrs, shutdownIn := ld.LaunchInputs(orc)
		assert.Equal(t, 1, len(inAddrs))
		conn, connErr := net.Dial("tcp", inAddrs[0])
		assert.Nil(t, connErr)

		oldMetricGatherer := ld.Loader.GetMetricGatherer()
		newMetricGatherer := ld.GetMetricGatherer()

		inCountCh := make(chan int, 1)
		outStatCh := launchOutputCounter(outBatchCh, inCountCh)

		// schedule reloading after 0.5 second
		go func() {
			time.Sleep(1 * time.Second)
			t.Run("reload errors", func(t *testing.T) {
				testReloadInvalidConfig(t, orc.(*ReloadableOrchestrator), func(newConf string) error {
					return ioutil.WriteFile(confFile.Name(), []byte(fmt.Sprintf(newConf, bufDir, srvAddr.String())), 0644)
				})
			})
			t.Run("reload normal", func(t *testing.T) {
				assert.Nil(t, ioutil.WriteFile(confFile.Name(), []byte(fmt.Sprintf(sampleYML2, bufDir, srvAddr.String())), 0644))
				orc.(*ReloadableOrchestrator).reload()
			})
		}()

		// keep sending logs for one second
		start := time.Now()
		numInput := 0
		for {
			if time.Since(start) >= 2*time.Second {
				break
			}
			numInput++
			_, sendErr := conn.Write([]byte(fmt.Sprintf(`<167>1 2020-07-20T03:48:20.154+03:00 host1 appServ/foo.com 51629 cron.log - Test me@gmail.com, %d
`, numInput)))
			assert.Nil(t, sendErr)
		}
		assert.Nil(t, conn.Close())

		inCountCh <- numInput
		outStat := <-outStatCh

		shutdownIn()
		orc.Shutdown()

		t.Run("compare counters", func(tt *testing.T) {
			t.Logf("sent %d, received old %d, received new %d, recoveried %d", numInput, outStat.NumOld, outStat.NumNew, outStat.NumRecovered)
			assert.Greater(tt, numInput, 0)
			assert.Greater(tt, outStat.NumOld, 0)
			assert.Greater(tt, outStat.NumNew, 0)
			assert.Equal(tt, numInput, outStat.NumOld+outStat.NumNew)
		})

		t.Run("check metrics before reloading", func(tt *testing.T) {
			metricFamilies, promErr := oldMetricGatherer.Gather()

			assert.Nil(tt, promErr)
			// input metrics are carried to new registry
			assert.Equal(tt, float64(outStat.NumOld), getMetricValue(t, metricFamilies, "process_passed_records_total", nil))
		})

		t.Run("check metrics after reloading", func(tt *testing.T) {
			metricFamilies, promErr := newMetricGatherer.Gather()

			assert.Nil(tt, promErr)
			assert.Equal(tt, float64(numInput), getMetricValue(t, metricFamilies, "input_passed_records_total", map[string]string{"protocol": "syslog"}))
			assert.Equal(tt, float64(outStat.NumNew), getMetricValue(t, metricFamilies, "process_passed_records_total", nil))

			labelledCounters := findMetricFamily(t, metricFamilies, "process_labelled_records_total")

			assert.Empty(tt, promext.MatchExportedMetrics(labelledCounters, map[string]string{"label": "emailFilter"}), "old label sets removed")
			assert.Equal(tt, float64(outStat.NumNew), promext.SumExportedMetrics(labelledCounters, map[string]string{"label": "MyMailFilter"}), "new label sets created and filled")

			// no failure because in testReloadInvalidConfig initiateReload() is called directly
			assert.Equal(tt, `slogagent_reloads_total{status="failure"} 0
slogagent_reloads_total{status="success"} 1
`, promext.DumpMetrics("slogagent_reloads_total", true, false, prometheus.DefaultGatherer))
		})

	})
}

func testReloadInvalidConfig(t *testing.T, orc *ReloadableOrchestrator, writeConf func(string) error) {
	{
		assert.Nil(t, writeConf(assembleConfig(
			strings.ReplaceAll(sampleSchemaConf, "maxFields: 12", "maxFields: 13"),
			sampleInputConf,
			sampleOrchestrationConf,
			sampleTransformationConf,
			sampleOutputConf,
		)))
		f, err := orc.initiateReload()
		assert.Nil(t, f)
		assert.EqualError(t, err, "schema/maxFields must not change: old=12, new=13")
	}
	{
		assert.Nil(t, writeConf(assembleConfig(
			strings.ReplaceAll(sampleSchemaConf, "facility, level, time,", "time, facility, level,"),
			sampleInputConf,
			sampleOrchestrationConf,
			sampleTransformationConf,
			sampleOutputConf,
		)))
		f, err := orc.initiateReload()
		assert.Nil(t, f)
		assert.EqualError(t, err, "schema/fields: required field \"facility\" has been moved, from=1th to=2th")
	}
	{
		assert.Nil(t, writeConf(assembleConfig(
			strings.ReplaceAll(sampleSchemaConf, ", extradata", ""),
			sampleInputConf,
			sampleOrchestrationConf,
			sampleTransformationConf,
			sampleOutputConf,
		)))
		f, err := orc.initiateReload()
		assert.Nil(t, f)
		assert.EqualError(t, err, "inputs[0] yaml line 9:5: incompatible with schema: field 'extradata' is not defined in schema")
	}
	{
		assert.Nil(t, writeConf(assembleConfig(
			sampleSchemaConf,
			sampleInputConf+`
      - type: delFields
        keys: [sn]
`,
			sampleOrchestrationConf,
			sampleTransformationConf,
			sampleOutputConf,
		)))
		f, err := orc.initiateReload()
		assert.Nil(t, f)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "inputs must not change: old=- type: syslog")
	}
	{
		assert.Nil(t, writeConf(assembleConfig(
			sampleSchemaConf,
			sampleInputConf,
			strings.ReplaceAll(sampleOrchestrationConf, "keys: [app]", "keys: [app, source]"),
			sampleTransformationConf,
			sampleOutputConf,
		)))
		f, err := orc.initiateReload()
		assert.Nil(t, f)
		assert.EqualError(t, err, "orchestration/keys must not change: old=[app], new=[app source]")
	}
}

type outputStatistics struct {
	NumRecovered int
	NumOld       int
	NumNew       int
}

func launchOutputCounter(logCh <-chan forwardprotocol.EventEntry, inputCompleted <-chan int) <-chan outputStatistics {
	outStatChan := make(chan outputStatistics, 1)

	go func() {
		outStat := outputStatistics{}
		lastSN := 0
		numInput := 0
		stopping := false
	LOOP:
		for {
			select {
			case log := <-logCh:
				newSN, snErr := strconv.Atoi(log.Record["sn"].(string))
				if snErr != nil {
					logger.Panicf("failed to parse SN: %s", log.Record["sn"])
				}
				switch {
				case newSN > lastSN+1:
					logger.Panicf("unexpected SN: %d, should be %d", newSN, lastSN+1)
				case newSN <= lastSN: // old logs resent from input to server
					outStat.NumRecovered++
					continue LOOP
				}
				lastSN = newSN

				content := log.Record["log"].(string)
				switch content {
				case "Test REDACTED hello":
					outStat.NumOld++
					if _, ok := log.Record["sn2"]; ok {
						logger.Panic("unexpected sn2 field: ", log.Record)
					}
				case "Test REDACTED new":
					outStat.NumNew++
					if log.Record["sn2"].(string) != "New "+log.Record["sn"].(string) {
						logger.Panic("unexpected sn2: ", log.Record["sn2"])
					}
				default:
					logger.Panic("unexpected log: ", log)
				}

				if stopping && numInput == outStat.NumOld+outStat.NumNew {
					logger.Info("output stopped")
					outStatChan <- outStat
					return
				}
			case numInput = <-inputCompleted:
				logger.Info("input done, stopping output")
				stopping = true
			}
		}
	}()

	return outStatChan
}
