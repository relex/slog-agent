package datadog

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/klauspost/compress/gzip"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/output/shared"
	"github.com/stretchr/testify/assert"
)

const (
	testChunkMaxSizeBytes = 1000
	testChunkMaxRecords   = 1000
	testBufCapacity       = 1000
	testChunkIDSuffix     = ".ddtest"
)

func countExpectedPayloadLength(payload string, writeIterations int) int {
	return len(payload)*writeIterations + len(",")*(writeIterations-1) + len("[]")
}

func TestDatadogOutput_Succeeds_OnGzippedInput(t *testing.T) {
	log := logger.Root()
	newChunkFunc := buildNewChunkFunc(log, testChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	payload := `testPayload`
	writeIterations := 5
	for i := 0; i < writeIterations; i++ {
		assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
	}

	chunk := packer.FlushBuffer()
	assert.True(t, strings.HasSuffix(chunk.ID, testChunkIDSuffix))

	reader, err := gzip.NewReader(bytes.NewReader(chunk.Data))
	assert.NoError(t, err)

	unzippedChunkData, err := io.ReadAll(reader)
	assert.NoError(t, err)

	assert.Contains(t, string(unzippedChunkData), payload)
	assert.Equal(t, countExpectedPayloadLength(payload, writeIterations), len(unzippedChunkData))
}

func TestDatadogOutput_Flushes_OnMaxRecordsReached(t *testing.T) {
	localChunkMaxRecords := 5

	log := logger.Root()
	newChunkFunc := buildNewChunkFunc(log, localChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	payload := base.LogStream("testPayload")
	writeIterations := 50

	for i := 0; i < writeIterations; i++ {
		if i != 0 && i%localChunkMaxRecords == 0 {
			chunk := packer.WriteStream(payload)
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, testChunkIDSuffix))

			reader, err := gzip.NewReader(bytes.NewReader(chunk.Data))
			assert.NoError(t, err)
			unzippedChunkData, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, countExpectedPayloadLength(string(payload), localChunkMaxRecords), len(unzippedChunkData))
		} else {
			assert.Nil(t, packer.WriteStream(payload))
		}
	}
}

func TestDatadogOutput_Flushes_OnMaxBytesReached(t *testing.T) {
	log := logger.Root()
	newChunkFunc := buildNewChunkFunc(log, testChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	payload := "10bytes..."
	iterationsTillOverflow := (testChunkMaxSizeBytes - len("[]")) / (len(payload) + len(","))

	for i := 0; i < iterationsTillOverflow*5; i++ {
		if i != 0 && i%iterationsTillOverflow == 0 {
			chunk := packer.WriteStream(base.LogStream(payload))
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, testChunkIDSuffix))
			reader, err := gzip.NewReader(bytes.NewReader(chunk.Data))
			assert.NoError(t, err)
			unzippedChunkData, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, countExpectedPayloadLength(string(payload), iterationsTillOverflow), len(unzippedChunkData))

		} else {
			if !assert.Nil(t, packer.WriteStream(base.LogStream(payload))) {
				continue
			}
		}
	}
}

func TestDatadogOutput_FlushNoPanic_OnNilCurrentChunk(t *testing.T) {
	log := logger.Root()
	newChunkFunc := buildNewChunkFunc(log, testChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	assert.Nil(t, packer.FlushBuffer())
}
