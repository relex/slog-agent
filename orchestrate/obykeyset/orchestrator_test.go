package obykeyset

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/stretchr/testify/assert"
)

func TestByKeySetOrchestrator(t *testing.T) {
	producerCount := 5
	tlogger := logger.WithField("test", t.Name())
	schema := base.MustNewLogSchema([]string{"level", "app", "msg"})
	collectedLogsByTag := make(map[string]*[]*base.LogRecord)
	launchWorkers := func(parentLogger logger.Logger, tag string, id string, input <-chan []*base.LogRecord, metricCreator promreg.MetricCreator, onStopped func()) {
		t.Logf("new worker %s: %s", tag, id)
		_, ok := collectedLogsByTag[tag]
		assert.False(t, ok, tag)
		collectedLogs := make([]*base.LogRecord, 0, 100)
		collectedLogsByTag[tag] = &collectedLogs
		go func() {
			counter := metricCreator.AddOrGetCounter("mycounter", "", nil, nil)
			for rec := range input {
				collectedLogs = append(collectedLogs, rec...)
				counter.Add(uint64(len(rec)))
			}
			onStopped()
		}()
	}
	mfactory := promreg.NewMetricFactory("testo_", nil, nil)
	orchestrator := NewOrchestrator(tlogger, schema, []string{"level", "app"}, "$level-$app", mfactory, launchWorkers, nil)
	producerWaiter := &sync.WaitGroup{}
	for i := 0; i < producerCount; i++ {
		producerWaiter.Add(1)
		go func(childNum int) {
			id := fmt.Sprintf("conn=%d", childNum)
			ch := orchestrator.NewSink(id, 123)
			ch.Accept([]*base.LogRecord{
				schema.NewTestRecord2(
					time.Unix(int64(childNum)*10+1, 0),
					base.LogFields{"error", "sshd", "auth " + id},
				),
				schema.NewTestRecord2(
					time.Unix(int64(childNum)*10+2, 0),
					base.LogFields{"info", "sshd", "login " + id},
				),
				schema.NewTestRecord2(
					time.Unix(int64(childNum)*10+3, 0),
					base.LogFields{"info", "lptd", "print " + id},
				),
				schema.NewTestRecord2(
					time.Unix(int64(childNum)*10+2, 0),
					base.LogFields{"info", "sshd", "logout " + id},
				),
			})
			ch.Close()
			producerWaiter.Done()
		}(i)
	}
	producerWaiter.Wait()
	orchestrator.Shutdown()

	assert.Equal(t, 3, len(collectedLogsByTag))
	if logs, ok := collectedLogsByTag["error-sshd"]; assert.True(t, ok) {
		assert.Equal(t, producerCount*1, len(*logs))
		for i, rec := range *logs {
			assert.Contains(t, rec.Fields[2], "auth ", i)
		}
	}
	if logs, ok := collectedLogsByTag["info-sshd"]; assert.True(t, ok) {
		assert.Equal(t, producerCount*2, len(*logs))
		for i, rec := range *logs {
			if i%2 == 0 {
				assert.Contains(t, rec.Fields[2], "login ", i)
			} else {
				assert.Contains(t, rec.Fields[2], "logout ", i)
			}
		}
	}
	if logs, ok := collectedLogsByTag["info-lptd"]; assert.True(t, ok) {
		assert.Equal(t, producerCount*1, len(*logs))
		for i, rec := range *logs {
			assert.Contains(t, rec.Fields[2], "print ", i)
		}
	}

	assert.Equal(t, `testo_process_mycounter{key_app="lptd",key_level="info",orchestrator="byKeySet"} 5
testo_process_mycounter{key_app="sshd",key_level="error",orchestrator="byKeySet"} 5
testo_process_mycounter{key_app="sshd",key_level="info",orchestrator="byKeySet"} 10
`, promext.DumpMetrics("", true, false, mfactory))
}
