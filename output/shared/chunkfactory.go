package shared

import (
	"bytes"
	"io"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

type (
	InitCompessorFunc func(log logger.Logger, w io.Writer) io.WriteCloser
	NewChunkFunc      func(log logger.Logger, id string, writeBuffer *bytes.Buffer, maxRecords, maxBytes int) Chunker

	Chunker interface {
		CanAppendData(dataLength int) bool
		FinalizeChunk() (*base.LogChunk, error)
		Write(data base.LogStream) error
	}
)

type IntermediateChunkFactory struct {
	log                  logger.Logger
	reusedChunkBuffer    *bytes.Buffer // since chunks are created consecutively, it's possible for them to share the data buffer
	idGenerator          *chunkIDGenerator
	newChunkFunc         NewChunkFunc
	maxRecords, maxBytes int
}

func NewChunkFactory(
	log logger.Logger,
	idSuffix string,
	bufCapacity int,
	newChunkFunc NewChunkFunc,
	maxRecords, maxBytes int,
) *IntermediateChunkFactory {
	return &IntermediateChunkFactory{
		log:               log,
		reusedChunkBuffer: bytes.NewBuffer(make([]byte, 0, bufCapacity)),
		idGenerator:       newChunkIDGenerator(idSuffix),
		newChunkFunc:      newChunkFunc,
		maxRecords:        maxRecords,
		maxBytes:          maxBytes,
	}
}

func (factory *IntermediateChunkFactory) NewChunk() Chunker {
	return factory.newChunkFunc(factory.log, factory.idGenerator.Generate(), factory.reusedChunkBuffer, factory.maxRecords, factory.maxBytes)
}
