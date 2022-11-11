package shared

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// messagePacker writes incoming log messages into a batch (chunk)
type messagePacker struct {
	logger       logger.Logger
	currentChunk Chunker // current chunk before being made into final message
	chunkFactory *IntermediateChunkFactory
}

// NewMessagePacker creates a LogChunkMaker to pack MessagePackEventStream(s) into Message(s)
// The resulting chunk itself can be saved on disk with ID as the filename, or send as request to upstream
func NewMessagePacker(log logger.Logger, chunkFactory *IntermediateChunkFactory) *messagePacker {
	return &messagePacker{
		logger:       log.WithField(defs.LabelComponent, "MessagePacker"),
		currentChunk: nil,
		chunkFactory: chunkFactory,
	}
}

func (packer *messagePacker) WriteStream(stream base.LogStream) *base.LogChunk {
	var previousChunk *base.LogChunk
	if packer.currentChunk != nil {
		if !packer.currentChunk.CanAppendData(len(stream)) {
			previousChunk = packer.FlushBuffer()
		}
	}

	if packer.currentChunk == nil {
		packer.currentChunk = packer.chunkFactory.NewChunk()
	}

	if err := packer.currentChunk.Write(stream); err != nil {
		packer.logger.Errorf("error writing data to chunk: %s", err)
	}

	return previousChunk
}

func (packer *messagePacker) FlushBuffer() *base.LogChunk {
	if packer.currentChunk == nil {
		return nil
	}

	result, err := packer.currentChunk.FinalizeChunk()
	if err != nil {
		packer.logger.Errorf("failed to make chunk: %s", err)
		return nil
	}

	packer.currentChunk = nil

	return result
}
