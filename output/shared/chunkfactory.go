package shared

import (
	"bytes"
	"io"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
)

type InitCompessorFunc func(log logger.Logger, w io.Writer) io.WriteCloser

type chunkEncoder interface {
	EncodeChunk(chunk *BasicChunk) ([]byte, error)
}

type BasicChunkFactory struct {
	log               logger.Logger
	reusedChunkBuffer *bytes.Buffer // since chunks are created consecutively, it's possible for them to share the data buffer
	initCompessor     InitCompessorFunc
	idGenerator       *chunkIDGenerator
	chunkEncoder      chunkEncoder
}

type BasicChunk struct {
	id           string
	numRecords   int
	numBytes     int
	compressor   io.WriteCloser // could be something like a gzip.Writer or nil to disable compression
	reusedBuffer *bytes.Buffer  // an actual buffer that compressor writes to
	encoder      chunkEncoder   // a function to finalize/encode the entire chunk after it's been assembled
}

func NewChunkFactory(log logger.Logger, idSuffix string, msgBufCapacity int, initCompessor InitCompessorFunc, encoder chunkEncoder) *BasicChunkFactory {
	return &BasicChunkFactory{
		log:               log,
		reusedChunkBuffer: bytes.NewBuffer(make([]byte, 0, msgBufCapacity)),
		initCompessor:     initCompessor,
		idGenerator:       newChunkIDGenerator(idSuffix),
		chunkEncoder:      encoder,
	}
}

func (factory *BasicChunkFactory) NewChunk() *BasicChunk {
	chunk := &BasicChunk{
		id:           factory.idGenerator.Generate(),
		numRecords:   0,
		numBytes:     0,
		compressor:   nil,
		reusedBuffer: factory.reusedChunkBuffer,
		encoder:      factory.chunkEncoder,
	}

	if factory.initCompessor != nil {
		chunk.compressor = factory.initCompessor(factory.log, factory.reusedChunkBuffer)
	}

	return chunk
}

// Write appends new log to log chunk
func (chunk *BasicChunk) Write(data []byte) error {
	var err error

	if chunk.compressor != nil {
		_, err = chunk.compressor.Write(data)
	} else {
		_, err = chunk.reusedBuffer.Write(data)
	}

	if err == nil {
		chunk.numRecords++
		chunk.numBytes += len(data)
	}

	return err
}

func (chunk *BasicChunk) FinalizeChunk() (*base.LogChunk, error) {
	if chunk.compressor != nil {
		if err := chunk.compressor.Close(); err != nil {
			return nil, err
		}
	}

	defer chunk.reusedBuffer.Reset()

	var chunkData []byte

	if chunk.encoder != nil {
		encodedChunk, err := chunk.encoder.EncodeChunk(chunk)
		if err != nil {
			return nil, err
		}
		chunkData = encodedChunk
	} else {
		chunkData = util.CopySlice(chunk.Bytes())
	}

	return &base.LogChunk{
		ID:    chunk.GetID(),
		Data:  chunkData,
		Saved: false,
	}, nil
}

func (chunk *BasicChunk) Bytes() []byte {
	return chunk.reusedBuffer.Bytes()
}

func (chunk *BasicChunk) GetID() string {
	return chunk.id
}

func (chunk *BasicChunk) GetNumRecords() int {
	return chunk.numRecords
}

func (chunk *BasicChunk) GetNumBytes() int {
	return chunk.numBytes
}

func (chunk *BasicChunk) IsCompressed() bool {
	return chunk.compressor != nil
}
