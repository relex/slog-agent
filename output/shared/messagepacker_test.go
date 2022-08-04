package shared

import (
	"bytes"
	"strings"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/stretchr/testify/assert"
)

var basicTestPackerConfig = PackerConfig{
	MsgBufCapacity:    1000,
	ChunkMaxSizeBytes: 1000,
	ChunkMaxRecords:   1000,
	ChunkIDSuffix:     ".test",
	UseCompression:    false,
}

type mockEncoder struct {
	buffer *bytes.Buffer
}

func (enc *mockEncoder) EncodeChunkAsMessage(data []byte, id string, numRecords, numBytes int, isCompressed bool) error {
	_, err := enc.buffer.Write(data)
	return err
}

func (enc *mockEncoder) GetEncodedResult() []byte { return enc.buffer.Bytes() }
func (enc *mockEncoder) ResetBuffer()             { enc.buffer.Reset() }

func TestMessagePacker_Succeeds_OnSimpleUncompressedInput(t *testing.T) {
	enc := &mockEncoder{buffer: bytes.NewBuffer(make([]byte, 0))}
	cfg := basicTestPackerConfig
	packer := NewMessagePacker(logger.Root(), &cfg, enc)

	payload := "testPayload"
	writeIterations := 5
	for i := 0; i < writeIterations; i++ {
		assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
	}

	chunk := packer.FlushBuffer()

	assert.True(t, strings.HasSuffix(chunk.ID, cfg.ChunkIDSuffix))
	assert.Equal(t, len(payload)*writeIterations, len(chunk.Data))
}

func TestMessagePacker_Flushes_OnMaxRecordsReached(t *testing.T) {
	enc := &mockEncoder{buffer: bytes.NewBuffer(make([]byte, 0))}
	cfg := basicTestPackerConfig
	cfg.ChunkMaxRecords = 5
	packer := NewMessagePacker(logger.Root(), &cfg, enc)

	payload := "testPayload"
	writeIterations := 50
	for i := 0; i < writeIterations; i++ {
		if i != 0 && i%cfg.ChunkMaxRecords == 0 {
			chunk := packer.WriteStream(base.LogStream(payload))
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, cfg.ChunkIDSuffix))
			assert.Equal(t, len(payload)*cfg.ChunkMaxRecords, len(chunk.Data))
		} else {
			assert.Nil(t, packer.WriteStream(base.LogStream(payload)))
		}
	}
}

func TestMessagePacker_Flushes_OnMaxBytesReached(t *testing.T) {
	enc := &mockEncoder{buffer: bytes.NewBuffer(make([]byte, 0))}
	cfg := basicTestPackerConfig
	packer := NewMessagePacker(logger.Root(), &cfg, enc)

	payload := "10bytes..."
	iterationsTillOverflow := cfg.ChunkMaxSizeBytes / len(payload)

	for i := 0; i < iterationsTillOverflow*5; i++ {
		if i != 0 && i%iterationsTillOverflow == 0 {
			chunk := packer.WriteStream(base.LogStream(payload))
			assert.NotNil(t, chunk)
			assert.True(t, strings.HasSuffix(chunk.ID, cfg.ChunkIDSuffix))
			assert.Equal(t, cfg.ChunkMaxSizeBytes, len(chunk.Data))
		} else {
			if !assert.Nil(t, packer.WriteStream(base.LogStream(payload))) {
				continue
			}
		}
	}
}

// func TestForwardMessageMaker(t *testing.T) {
// 	chunkMaker := NewMessagePacker(logger.Root(), "hello", forwardprotocol.ModeCompressedPackedForward)
// 	for i, record := range testInputRecords {
// 		stream := serializer.SerializeRecord(testSchema.CopyTestRecord(record))
// 		assert.Nil(t, chunkMaker.WriteStream(stream), i)
// 	}

// 	chunk := chunkMaker.FlushBuffer()
// 	assert.NotNil(t, chunk)
// 	assert.NotEmpty(t, chunk.ID)

// 	chunkDecoder := msgpack.NewDecoder(bytes.NewBuffer(chunk.Data))
// 	var message forwardprotocol.Message
// 	assert.Nil(t, chunkDecoder.Decode(&message))
// 	assert.Equal(t, "hello", message.Tag)
// 	assert.Equal(t, chunk.ID, message.Option.Chunk)
// 	assert.Equal(t, len(testInputRecords), message.Option.Size)
// 	assert.Equal(t, forwardprotocol.CompressionFormat, message.Option.Compressed)
// 	assert.Equal(t, len(testInputRecords), len(message.Entries))

// 	for i := range testOutputFieldMaps {
// 		assert.Equal(t, testInputRecords[i].Timestamp.UnixNano(), message.Entries[i].Time.UnixNano(), fmt.Sprintf("record[%d] timestamp", i))
// 		assert.Equal(t, testOutputFieldMaps[i], message.Entries[i].Record, fmt.Sprintf("record[%d] fields", i))
// 	}
// }

// func TestForwardMessageMakerJoining(t *testing.T) {
// 	// test records of 120 bytes each
// 	inputRecords := generateMassTestLogRecords(100)
// 	// generate stream parts, each should be 360 bytes
// 	serializer, serr := NewEventSerializer(logger.Root(), testSchema, testSerializationConfig)
// 	assert.Nil(t, serr)
// 	oldChunkLimit := outputChunkMaxDataBytes
// 	outputChunkMaxDataBytes = 3000
// 	chunkMaker := NewMessagePacker(logger.Root(), "hello", forwardprotocol.ModeCompressedPackedForward)
// 	hasPrevChunk := true
// 	for i, record := range inputRecords {
// 		stream := serializer.SerializeRecord(record)
// 		maybeChunk := chunkMaker.WriteStream(stream)
// 		if hasPrevChunk {
// 			// two chunks in row shouldn't happen
// 			assert.Nil(t, maybeChunk, "stream[%d] => chunk", i)
// 		}
// 		if maybeChunk != nil {
// 			hasPrevChunk = true
// 			// for best-speed compression
// 			assert.GreaterOrEqual(t, len(maybeChunk.Data), 200, fmt.Sprintf("stream[%d] => chunk", i))
// 			assert.LessOrEqual(t, len(maybeChunk.Data), 300, fmt.Sprintf("stream[%d] => chunk", i))
// 		} else {
// 			hasPrevChunk = false
// 		}
// 	}
// 	lastChunk := chunkMaker.FlushBuffer()
// 	assert.NotNil(t, lastChunk, "chunk[last]")
// 	outputChunkMaxDataBytes = oldChunkLimit
// }

// func generateMassTestLogRecords(count int) []*base.LogRecord {
// 	inputRecords := make([]*base.LogRecord, count)
// 	for i := 0; i < count; i++ {
// 		inputRecords[i] = testSchema.NewTestRecord1(
// 			base.LogFields{
// 				"",
// 				"",
// 				"Hello [" + strings.Repeat("0123456789", 5) + "] " + fmt.Sprintf("%010d", i),
// 				"",
// 				fmt.Sprintf("K%d", i%10),
// 			},
// 		)
// 	}
// 	// serialized bytes per record: 120
// 	return inputRecords
// }
