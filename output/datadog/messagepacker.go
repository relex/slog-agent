package datadog

import (
	"bytes"

	"github.com/klauspost/compress/gzip"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/output/shared"
	"github.com/relex/slog-agent/util"
)

const (
	gzipCompressionLevel = gzip.BestSpeed
	chunkIDSuffix        = ".dd" // output-specific file extension for generated chunks
)

// outputChunkMaxDataBytes defines the max uncompressed data size of a LogChunk.
//
// see the api docs here: https://docs.datadoghq.com/api/latest/logs/#send-logs
var outputChunkMaxDataBytes = 5 * 1024 * 1024

// outputChunkMaxRecords defined the maximum amount of log entries in a single chunk for datadog output
var outputChunkMaxRecords = 1000

type messagePacker struct {
	logger              logger.Logger
	currentChunk        *forwardMessageOpenChunk // current chunk before being made into final message
	reusedGzipBuffer    *bytes.Buffer            // buffer for gzipWriter for log records
	reusedMessageBuffer *bytes.Buffer            // buffer for final message
	chunkIDGenerator    *shared.ChunkIDGenerator
	useCompression      bool
}

type forwardMessageOpenChunk struct {
	gzipWriter     *gzip.Writer // gzip writer can be nil if no compression, write to reusedGzipBuffer
	id             string       // chunk ID
	numRecords     int          // numbers of collected log records
	numStreamBytes int          // uncompressed length of written stream data
}

func NewMessagePacker(parentLogger logger.Logger) *messagePacker {
	return &messagePacker{
		logger:              parentLogger.WithField(defs.LabelComponent, "DatadogMessagePacker"),
		currentChunk:        nil,
		reusedGzipBuffer:    bytes.NewBuffer(make([]byte, 0, 1*1024*1024)),
		reusedMessageBuffer: bytes.NewBuffer(make([]byte, 0, 1*1024*1024)),
		chunkIDGenerator:    shared.NewChunkIDGenerator(chunkIDSuffix),
		useCompression:      true,
	}
}

func (packer *messagePacker) WriteStream(stream base.LogStream) *base.LogChunk {
	var previousChunk *base.LogChunk
	if packer.currentChunk != nil &&
		(packer.currentChunk.numRecords >= outputChunkMaxRecords || packer.currentChunk.numStreamBytes+len(stream) > outputChunkMaxDataBytes) {
		previousChunk = packer.FlushBuffer()
	}
	packer.ensureOpenChunk()
	if packer.currentChunk.gzipWriter != nil {
		if _, err := packer.currentChunk.gzipWriter.Write(stream); err != nil {
			packer.logger.Errorf("error writing to gzip writer: %s", err.Error())
		}
	} else if _, err := packer.reusedGzipBuffer.Write(stream); err != nil {
		packer.logger.Errorf("error writing to buffer: %s", err.Error())
	}
	packer.currentChunk.numRecords++
	packer.currentChunk.numStreamBytes += len(stream)
	return previousChunk
}

func (packer *messagePacker) FlushBuffer() *base.LogChunk {
	if packer.currentChunk == nil {
		return nil
	}
	openChunk := *packer.currentChunk
	packer.currentChunk = nil
	if openChunk.gzipWriter != nil {
		if err := openChunk.gzipWriter.Close(); err != nil {
			packer.logger.Errorf("failed to close gzip writer: %s", err.Error())
		}
	}
	var maybeChunk *base.LogChunk
	_, err := packer.reusedMessageBuffer.Write(packer.reusedGzipBuffer.Bytes())
	if err != nil {
		packer.logger.Errorf("failed to make chunk: %s", err.Error())
	} else {
		maybeChunk = &base.LogChunk{
			ID:    openChunk.id,
			Data:  util.CopySlice(packer.reusedMessageBuffer.Bytes()),
			Saved: false,
		}
	}
	packer.reusedGzipBuffer.Reset()
	packer.reusedMessageBuffer.Reset()
	return maybeChunk
}

func (packer *messagePacker) ensureOpenChunk() {
	if packer.currentChunk != nil {
		return
	}
	packer.currentChunk = &forwardMessageOpenChunk{
		id:             packer.chunkIDGenerator.Generate(),
		gzipWriter:     packer.createGzipWriter(),
		numRecords:     0,
		numStreamBytes: 0,
	}
}

func (packer *messagePacker) createGzipWriter() *gzip.Writer {
	if !packer.useCompression {
		return nil
	}
	writer, err := gzip.NewWriterLevel(packer.reusedGzipBuffer, gzipCompressionLevel)
	if err != nil {
		packer.logger.Errorf("failed to create GzipWriter: %s", err)
	}
	return writer
}
