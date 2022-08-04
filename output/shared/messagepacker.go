package shared

import (
	"bytes"
	"compress/gzip"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// gzipCompressionLevel for ForwardMessage.
// BestSpeed uses 30% more space and roughly same percentage in time saving
const gzipCompressionLevel = gzip.BestSpeed

type PackerConfig struct {
	MsgBufCapacity    int
	ChunkMaxSizeBytes int
	ChunkMaxRecords   int
	ChunkIDSuffix     string
	UseCompression    bool
}

// messagePacker writes incoming log messages into a batch (chunk) and encodes them to gzip
type messagePacker struct {
	logger            logger.Logger
	useCompression    bool
	reusedGzipBuffer  *bytes.Buffer // buffer for gzipWriter for log records
	chunkIDGenerator  *chunkIDGenerator
	currentChunk      *forwardMessageOpenChunk // current chunk before being made into final message
	chunkMaxSizeBytes int
	chunkMaxRecords   int
	encoder           messageEncoder // output-specific chunk encoder
}

type forwardMessageOpenChunk struct {
	ID             string       // chunk ID
	GZIPWriter     *gzip.Writer // gzip writer can be nil if no compression, write to reusedGzipBuffer
	NumRecords     int          // numbers of collected log records
	NumStreamBytes int          // uncompressed length of written stream data
}

type messageEncoder interface {
	EncodeChunkAsMessage(data []byte, id string, numRecords, numBytes int, isCompressed bool) error
	GetEncodedResult() []byte
	ResetBuffer()
}

// NewMessagePacker creates a LogChunkMaker to pack MessagePackEventStream(s) into Message(s)
// The resulting chunk itself can be saved on disk with ID as the filename, or send as request to upstream
func NewMessagePacker(
	parentLogger logger.Logger,
	cfg *PackerConfig,
	encoder messageEncoder,
) base.LogChunkMaker {
	return &messagePacker{
		logger:            parentLogger.WithField(defs.LabelComponent, "FluentdForwardMessagePacker"),
		useCompression:    cfg.UseCompression,
		reusedGzipBuffer:  bytes.NewBuffer(make([]byte, 0, cfg.MsgBufCapacity)),
		chunkIDGenerator:  newChunkIDGenerator(cfg.ChunkIDSuffix),
		currentChunk:      nil,
		chunkMaxSizeBytes: cfg.ChunkMaxSizeBytes,
		chunkMaxRecords:   cfg.ChunkMaxRecords,
		encoder:           encoder,
	}
}

func (packer *messagePacker) WriteStream(stream base.LogStream) *base.LogChunk {
	var previousChunk *base.LogChunk
	if packer.currentChunk != nil {
		if packer.currentChunk.NumRecords > 0 && packer.chunkMaxRecords >= packer.currentChunk.NumRecords {
			// flush when the amount of log records reaches max permitted amount, if it is defined
			previousChunk = packer.FlushBuffer()
		} else if packer.currentChunk.NumStreamBytes+len(stream) >= packer.chunkMaxSizeBytes {
			// otherwise flush when the total size reaches max permitted amount
			previousChunk = packer.FlushBuffer()
		}
	}

	packer.ensureOpenChunk()
	if packer.currentChunk.GZIPWriter != nil {
		if _, err := packer.currentChunk.GZIPWriter.Write(stream); err != nil {
			packer.logger.Errorf("error writing to gzip writer: %s", err.Error())
		}
	} else if _, err := packer.reusedGzipBuffer.Write(stream); err != nil {
		packer.logger.Errorf("error writing to buffer: %s", err.Error())
	}
	packer.currentChunk.NumRecords++
	packer.currentChunk.NumStreamBytes += len(stream)
	return previousChunk
}

func (packer *messagePacker) FlushBuffer() *base.LogChunk {
	if packer.currentChunk == nil {
		return nil
	}
	openChunk := *packer.currentChunk
	packer.currentChunk = nil
	if openChunk.GZIPWriter != nil {
		if err := openChunk.GZIPWriter.Close(); err != nil {
			packer.logger.Errorf("failed to close gzip writer: %s", err.Error())
		}
	}
	var maybeChunk *base.LogChunk
	err := packer.encoder.EncodeChunkAsMessage(packer.reusedGzipBuffer.Bytes(), openChunk.ID, openChunk.NumRecords, openChunk.NumStreamBytes, openChunk.GZIPWriter != nil)
	if err != nil {
		packer.logger.Errorf("failed to make chunk: %s", err.Error())
	} else {
		maybeChunk = &base.LogChunk{
			ID:    openChunk.ID,
			Data:  util.CopySlice(packer.encoder.GetEncodedResult()),
			Saved: false,
		}
	}
	packer.reusedGzipBuffer.Reset()
	packer.encoder.ResetBuffer()
	return maybeChunk
}

func (packer *messagePacker) ensureOpenChunk() {
	if packer.currentChunk != nil {
		return
	}
	packer.currentChunk = &forwardMessageOpenChunk{
		ID:             packer.chunkIDGenerator.generate(),
		GZIPWriter:     packer.createGzipWriter(),
		NumRecords:     0,
		NumStreamBytes: 0,
	}
}

func (packer *messagePacker) createGzipWriter() *gzip.Writer {
	if !packer.useCompression {
		return nil
	}
	gzWriter, gzErr := gzip.NewWriterLevel(packer.reusedGzipBuffer, gzipCompressionLevel)
	if gzErr != nil {
		packer.logger.Errorf("failed to create GzipWriter: %s", gzErr.Error())
	}
	return gzWriter
}
