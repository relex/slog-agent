package fluentdforward

import (
	"bytes"
	"encoding/json"
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

type mockJSONEncoder struct{ msgKey string }

func (enc *mockJSONEncoder) EncodeChunk(data []byte, params *encodeChunkParams) ([]byte, error) {
	return json.Marshal(map[string][]byte{enc.msgKey: data})
}

func TestFluentforwardOutput_Succeeds_OnUncompressedUnencodedInput(t *testing.T) {
	log := logger.Root()
	newChunkFunc := buildNewChunkFunc(log, nil, nil, testChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	payload := `testPayload`
	writeIterations := 5
	for i := 0; i < writeIterations; i++ {
		assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
	}

	chunk := packer.FlushBuffer()
	assert.True(t, strings.HasSuffix(chunk.ID, testChunkIDSuffix))

	assert.Contains(t, string(chunk.Data), payload)
	assert.Equal(t, len(payload)*writeIterations, len(chunk.Data))
}

func TestFluentforwardOutput_Succeeds_OnGzippedEncodedInput(t *testing.T) {
	log := logger.Root()
	encoder := &mockJSONEncoder{msgKey: "key"}
	newChunkFunc := buildNewChunkFunc(log, shared.InitGzipCompessor, encoder, testChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	payload := `testPayload`
	writeIterations := 5
	for i := 0; i < writeIterations; i++ {
		assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
	}

	chunk := packer.FlushBuffer()
	assert.True(t, strings.HasSuffix(chunk.ID, testChunkIDSuffix))

	var outputMap map[string][]byte
	err := json.Unmarshal(chunk.Data, &outputMap)
	assert.NoError(t, err)

	reader, err := gzip.NewReader(bytes.NewReader(outputMap[encoder.msgKey]))
	assert.NoError(t, err)

	unzippedChunkData, err := io.ReadAll(reader)
	assert.NoError(t, err)

	assert.Contains(t, string(unzippedChunkData), payload)
	assert.Equal(t, len(payload)*writeIterations, len(unzippedChunkData))
}

func TestFluentforwardOutput_Flushes_OnMaxRecordsReached(t *testing.T) {
	localChunkMaxRecords := 5

	log := logger.Root()
	encoder := &mockJSONEncoder{msgKey: "key"}
	newChunkFunc := buildNewChunkFunc(log, shared.InitGzipCompessor, encoder, localChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	payload := base.LogStream("testPayload")
	writeIterations := 50

	for i := 0; i < writeIterations; i++ {
		if i != 0 && i%localChunkMaxRecords == 0 {
			chunk := packer.WriteStream(payload)
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, testChunkIDSuffix))

			var outputMap map[string][]byte
			err := json.Unmarshal(chunk.Data, &outputMap)
			assert.NoError(t, err)

			reader, err := gzip.NewReader(bytes.NewReader(outputMap[encoder.msgKey]))
			assert.Nil(t, err)
			unzippedChunkData, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, len(payload)*localChunkMaxRecords, len(unzippedChunkData))
		} else {
			assert.Nil(t, packer.WriteStream(payload))
		}
	}
}

func TestFluentforwardOutput_Flushes_OnMaxBytesReached(t *testing.T) {
	log := logger.Root()
	newChunkFunc := buildNewChunkFunc(log, shared.InitGzipCompessor, nil, testChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	payload := "10bytes..."
	iterationsTillOverflow := testChunkMaxSizeBytes / len(payload)

	for i := 0; i < iterationsTillOverflow*5; i++ {
		if i != 0 && i%iterationsTillOverflow == 0 {
			chunk := packer.WriteStream(base.LogStream(payload))
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, testChunkIDSuffix))

			reader, err := gzip.NewReader(bytes.NewReader(chunk.Data))
			assert.Nil(t, err)
			unzippedChunkData, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, len(payload)*iterationsTillOverflow, len(unzippedChunkData))

		} else {
			if !assert.Nil(t, packer.WriteStream(base.LogStream(payload))) {
				continue
			}
		}
	}
}

func TestFluentforwardOutput_FlushNoPanic_OnNilCurrentChunk(t *testing.T) {
	log := logger.Root()
	encoder := newEncoder("tag", false, msgBufCapacity)
	newChunkFunc := buildNewChunkFunc(log, shared.InitGzipCompessor, encoder, testChunkMaxRecords, testChunkMaxSizeBytes)
	factory := shared.NewChunkFactory(testChunkIDSuffix, testBufCapacity, newChunkFunc)
	packer := shared.NewMessagePacker(log, factory)

	assert.Nil(t, packer.FlushBuffer())
}
