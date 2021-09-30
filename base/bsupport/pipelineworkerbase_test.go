package bsupport

import (
	"testing"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

type testPipelineWorker struct {
	PipelineWorkerBaseForLogRecords
	outputChannel chan []*base.LogRecord
}

func newTestPipelineWorker(parentLogger logger.Logger, input <-chan []*base.LogRecord) *testPipelineWorker {
	worker := &testPipelineWorker{
		PipelineWorkerBaseForLogRecords: NewPipelineWorkerBaseForLogRecords(
			parentLogger.WithField(defs.LabelComponent, "TestPipelineWorker"),
			input,
			false,
		),
		outputChannel: make(chan []*base.LogRecord, defs.IntermediateBufferedChannelSize),
	}
	worker.InitInternal(worker.onInput, nil, worker.onStop)
	return worker
}

func (worker *testPipelineWorker) onInput(inBuffer []*base.LogRecord, timeout <-chan time.Time) {
	outBuffer := make([]*base.LogRecord, 0, len(inBuffer))

	for index, record := range inBuffer {
		// skip the first record in buffer
		if index == 0 {
			continue
		}
		// add timestamp by 1
		record.Timestamp = record.Timestamp.Add(1 * time.Second)

		outBuffer = append(outBuffer, record)
	}

	select {
	case worker.outputChannel <- outBuffer:
		break
	case <-timeout:
		worker.Logger().Errorf("BUG: timeout sending to channel: %d records. stack=%s", len(outBuffer), util.Stack())
		break
	}
}

func (worker *testPipelineWorker) onStop(timeout <-chan time.Time) {
	close(worker.outputChannel)
	worker.Logger().Infof("destroy channel, remaining=%d", len(worker.outputChannel))
}

func TestPipelineWorkerBase(t *testing.T) {
	tlogger := logger.WithField("test", t.Name())
	input := make(chan []*base.LogRecord, 10)
	worker := newTestPipelineWorker(tlogger, input)
	worker.Launch()

	input <- []*base.LogRecord{
		// first record is to be skipped
		{
			Timestamp: time.Unix(30, 0),
			Fields:    base.LogFields{"yes"},
		},
		// rest of records are to have timestamp++
		{
			Timestamp: time.Unix(10, 0),
			Fields:    base.LogFields{"no"},
		},
	}
	input <- []*base.LogRecord{
		// first record is to be skipped
		{
			Timestamp: time.Unix(20, 0),
			Fields:    base.LogFields{"yes"},
		},
		// rest of records are to have timestamp++
		{
			Timestamp: time.Unix(40, 0),
			Fields:    base.LogFields{"yes"},
		},
	}

	{
		r1 := readFromTestWorker(worker)
		if assert.Equal(t, 1, len(r1)) {
			assert.Equal(t, "no", r1[0].Fields[0])
			assert.Equal(t, int64(11), r1[0].Timestamp.Unix())
		}
	}
	{
		r2 := readFromTestWorker(worker)
		if assert.Equal(t, 1, len(r2)) {
			assert.Equal(t, "yes", r2[0].Fields[0])
			assert.Equal(t, int64(41), r2[0].Timestamp.Unix())
		}
	}

	close(input)
	assert.True(t, worker.Stopped().Wait(defs.TestReadTimeout))
}

func TestPipelineWorkerBaseStop(t *testing.T) {
	oldChannelSize := defs.IntermediateBufferedChannelSize
	defs.IntermediateBufferedChannelSize = 10 // keep last logs in channel buffer
	tlogger := logger.WithField("test", t.Name())
	input := make(chan []*base.LogRecord, 10)
	worker := newTestPipelineWorker(tlogger, input)
	worker.Launch()
	input <- []*base.LogRecord{{}, {
		Timestamp: time.Unix(20, 0),
		Fields:    base.LogFields{"yes"},
	}}
	close(input)
	assert.True(t, worker.Stopped().Wait(defs.TestReadTimeout))
	{
		r1 := readFromTestWorker(worker)
		if assert.Equal(t, 1, len(r1)) {
			assert.Equal(t, "yes", r1[0].Fields[0])
			assert.Equal(t, int64(21), r1[0].Timestamp.Unix())
		}
	}
	defs.IntermediateBufferedChannelSize = oldChannelSize
}

func readFromTestWorker(worker *testPipelineWorker) []*base.LogRecord {
	select {
	case r := <-worker.outputChannel:
		return r
	case <-time.After(defs.TestReadTimeout):
		return nil
	}
}
