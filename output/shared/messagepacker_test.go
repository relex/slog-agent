package shared

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/stretchr/testify/assert"
)

const (
	chunkMaxSizeBytes = 1000
	chunkMaxRecords   = 1000
	msgBufCapacity    = 1000
	chunkIDSuffix     = ".test"
)

type mockJSONEncoder struct{ msgKey string }

func (enc *mockJSONEncoder) EncodeChunk(chunk *BasicChunk) ([]byte, error) {
	return json.Marshal(map[string][]byte{enc.msgKey: chunk.Bytes()})
}

func TestMessagePacker_Succeeds_OnUnencodedUncompressedInput(t *testing.T) {
	log := logger.Root()
	factory := NewChunkFactory(log, chunkIDSuffix, msgBufCapacity, nil, nil)
	packer := NewMessagePacker(log, chunkMaxSizeBytes, chunkMaxRecords, factory)

	payload := "testPayload"
	writeIterations := 5
	for i := 0; i < writeIterations; i++ {
		assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
	}

	chunk := packer.FlushBuffer()

	assert.True(t, strings.HasSuffix(chunk.ID, chunkIDSuffix))
	assert.Equal(t, len(payload)*writeIterations, len(chunk.Data))
}

func TestMessagePacker_Succeeds_OnEncodedGzippedInput(t *testing.T) {
	log := logger.Root()
	enc := &mockJSONEncoder{msgKey: "testMsg"}
	factory := NewChunkFactory(log, chunkIDSuffix, msgBufCapacity, InitGZIPCompessor, enc)
	packer := NewMessagePacker(log, chunkMaxSizeBytes, chunkMaxRecords, factory)

	payload := "testPayload"
	writeIterations := 5
	for i := 0; i < writeIterations; i++ {
		assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
	}

	chunk := packer.FlushBuffer()
	assert.True(t, strings.HasSuffix(chunk.ID, chunkIDSuffix))

	decodedMap := make(map[string][]byte)
	assert.NoError(t, json.Unmarshal(chunk.Data, &decodedMap))

	reader, err := gzip.NewReader(bytes.NewReader(decodedMap[enc.msgKey]))
	assert.NoError(t, err)

	unzippedChunkData, err := io.ReadAll(reader)
	assert.NoError(t, err)

	assert.Contains(t, string(unzippedChunkData), payload)
	assert.Equal(t, len(payload)*writeIterations, len(unzippedChunkData))
}

func TestMessagePacker_Flushes_OnMaxRecordsReached(t *testing.T) {
	localChunkMaxRecords := 5

	log := logger.Root()
	factory := NewChunkFactory(log, chunkIDSuffix, msgBufCapacity, nil, nil)
	packer := NewMessagePacker(log, chunkMaxSizeBytes, localChunkMaxRecords, factory)

	payload := "testPayload"
	writeIterations := 50
	for i := 0; i < writeIterations; i++ {
		if i != 0 && i%localChunkMaxRecords == 0 {
			chunk := packer.WriteStream(base.LogStream(payload))
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, chunkIDSuffix))
			assert.Equal(t, len(payload)*localChunkMaxRecords, len(chunk.Data))
		} else {
			assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
		}
	}
}

func TestMessagePacker_Flushes_OnMaxBytesReached(t *testing.T) {
	log := logger.Root()
	factory := NewChunkFactory(log, chunkIDSuffix, msgBufCapacity, nil, nil)
	packer := NewMessagePacker(log, chunkMaxSizeBytes, chunkMaxRecords, factory)

	payload := "10bytes..."
	iterationsTillOverflow := chunkMaxSizeBytes / len(payload)

	for i := 0; i < iterationsTillOverflow*5; i++ {
		if i != 0 && i%iterationsTillOverflow == 0 {
			chunk := packer.WriteStream(base.LogStream(payload))
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, chunkIDSuffix))
			assert.Equal(t, chunkMaxSizeBytes, len(chunk.Data))
		} else {
			if !assert.Nil(t, packer.WriteStream(base.LogStream(payload))) {
				continue
			}
		}
	}
}

func TestMessagePacker_FlushNoPanic_OnNilCurrentChunk(t *testing.T) {
	log := logger.Root()
	factory := NewChunkFactory(log, chunkIDSuffix, msgBufCapacity, nil, nil)
	packer := NewMessagePacker(log, chunkMaxSizeBytes, chunkMaxRecords, factory)

	assert.Nil(t, packer.FlushBuffer())
}
