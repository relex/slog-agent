package shared

import (
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

func TestMessagePacker_Succeeds_OnSimpleUncompressedInput(t *testing.T) {
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
