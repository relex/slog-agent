package test

import (
	"bytes"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

type inMemoryChunkSaver struct {
	logger logger.Logger
	buffer *bytes.Buffer
}

func newInMemoryChunkSaver(parentLogger logger.Logger) (chunkSaver, func() string) {
	buffer := &bytes.Buffer{}
	buffer.WriteString("[\n")
	return &inMemoryChunkSaver{
		logger: parentLogger,
		buffer: buffer,
	}, buffer.String
}

func (s *inMemoryChunkSaver) Write(chunk base.LogChunk, decoder base.ChunkDecoder) {
	_, err := decoder.DecodeChunkToJSON(chunk, []byte(",\n"), true, s.buffer)
	if err != nil {
		logger.Panicf("error writing: %s", err.Error())
	}
}

func (s *inMemoryChunkSaver) Close() {
	if _, err := s.buffer.WriteString("\n]\n"); err != nil {
		logger.Panicf("error writing JSON end: %s", err.Error())
	}
}
