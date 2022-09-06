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
	EncodeChunk(data []byte, params *EncodeChunkParams) ([]byte, error)
}

type EncodeChunkParams struct {
	ID           string
	NumRecords   int
	NumBytes     int
	IsCompressed bool
}

type IntermediateChunkFactory struct {
	log               logger.Logger
	reusedChunkBuffer *bytes.Buffer // since chunks are created consecutively, it's possible for them to share the data buffer
	initCompressor    InitCompessorFunc
	idGenerator       *chunkIDGenerator
	chunkEncoder      chunkEncoder
}

type IntermediateChunk struct {
	id           string
	numRecords   int
	numBytes     int
	compressor   io.WriteCloser // could be something like a gzip.Writer or nil to disable compression
	reusedBuffer *bytes.Buffer  // an actual buffer that compressor writes to
	encoder      chunkEncoder   // a function to finalize/encode the entire chunk after it's been assembled
}

func NewChunkFactory(log logger.Logger, idSuffix string, bufCapacity int, initCompressor InitCompessorFunc, encoder chunkEncoder) *IntermediateChunkFactory {
	return &IntermediateChunkFactory{
		log:               log,
		reusedChunkBuffer: bytes.NewBuffer(make([]byte, 0, bufCapacity)),
		initCompressor:    initCompressor,
		idGenerator:       newChunkIDGenerator(idSuffix),
		chunkEncoder:      encoder,
	}
}

func (factory *IntermediateChunkFactory) NewChunk() *IntermediateChunk {
	chunk := &IntermediateChunk{
		id:           factory.idGenerator.Generate(),
		numRecords:   0,
		numBytes:     0,
		compressor:   nil,
		reusedBuffer: factory.reusedChunkBuffer,
		encoder:      factory.chunkEncoder,
	}

	if factory.initCompressor != nil {
		chunk.compressor = factory.initCompressor(factory.log, factory.reusedChunkBuffer)
	}

	return chunk
}

// Write appends new log to log chunk
func (chunk *IntermediateChunk) Write(data base.LogStream) error {
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

func (chunk *IntermediateChunk) FinalizeChunk() (*base.LogChunk, error) {
	if chunk.compressor != nil {
		if err := chunk.compressor.Close(); err != nil {
			return nil, err
		}
	}

	defer chunk.reusedBuffer.Reset()

	var chunkData []byte

	if chunk.encoder != nil {
		encodeParams := &EncodeChunkParams{
			ID:           chunk.id,
			NumRecords:   chunk.numRecords,
			NumBytes:     chunk.numBytes,
			IsCompressed: chunk.compressor != nil,
		}
		encodedChunk, err := chunk.encoder.EncodeChunk(chunk.reusedBuffer.Bytes(), encodeParams)
		if err != nil {
			return nil, err
		}
		chunkData = encodedChunk
	} else {
		chunkData = util.CopySlice(chunk.reusedBuffer.Bytes())
	}

	return &base.LogChunk{
		ID:    chunk.id,
		Data:  chunkData,
		Saved: false,
	}, nil
}
