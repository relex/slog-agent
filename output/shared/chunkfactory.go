package shared

import (
	"bytes"
	"io"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

type (
	InitCompressorFunc func(log logger.Logger, w io.Writer) io.WriteCloser
	NewChunkFunc       func(id string, writeBuffer *bytes.Buffer) Chunker

	Chunker interface {
		CanAppendData(dataLength int) bool
		FinalizeChunk() (*base.LogChunk, error)
		Write(data base.LogStream) error
	}
)

type IntermediateChunkFactory struct {
	reusedChunkBuffer *bytes.Buffer // since chunks are created consecutively, it's possible for them to share the data buffer
	idGenerator       *chunkIDGenerator
	newChunkFunc      NewChunkFunc
}

func NewChunkFactory(
	idSuffix string,
	bufCapacity int,
	newChunkFunc NewChunkFunc,
) *IntermediateChunkFactory {
	return &IntermediateChunkFactory{
		reusedChunkBuffer: bytes.NewBuffer(make([]byte, 0, bufCapacity)),
		idGenerator:       newChunkIDGenerator(idSuffix),
		newChunkFunc:      newChunkFunc,
	}
}

func (factory *IntermediateChunkFactory) NewChunk() Chunker {
	return factory.newChunkFunc(factory.idGenerator.Generate(), factory.reusedChunkBuffer)
}
