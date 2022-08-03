package fluentdforward

import (
	"bytes"

	"github.com/klauspost/compress/gzip"
	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
	"github.com/vmihailenco/msgpack/v4"
)

// compressedMessage enables compression - needs to be supported by upstream (e.g. fluentd)
// The ratio is very high (20-50x) and critical for disk space and network bandwidth
// DO NOT disable compression except for uncommitted testing
const compressedMessage = true

// gzipCompressionLevel for ForwardMessage
// BestSpeed uses 30% more space and roughly same percentage in time saving
const gzipCompressionLevel = gzip.BestSpeed

// messageBufferCapacity is the initial capacity for buffers used for chunk and compression
// It only needs to be large enough to contain the largest compressed message
const messageBufferCapacity = 1 * 1024 * 1024

// outputChunkMaxDataBytes defines the max uncompressed data size of a LogChunk, not including necessary headers.
// The value must be well below Fluentd's Fluent::Plugin::Buffer::DEFAULT_CHUNK_LIMIT_SIZE, as some buffers are implicitly inserted and non-configurable.
//
// https://github.com/fluent/fluentd/blob/master/lib/fluent/plugin/buffer.rb#L39
var outputChunkMaxDataBytes = 7 * 1024 * 1024

type messagePacker struct {
	logger               logger.Logger
	tag                  string
	asArray              bool
	compressed           bool
	reusedGzipBuffer     *bytes.Buffer            // buffer for gzipWriter for log records
	reusedMessageBuffer  *bytes.Buffer            // buffer for final message
	reusedMessageEncoder *msgpack.Encoder         // encoder for final message
	currentChunk         *forwardMessageOpenChunk // current chunk before being made into final message
}

type forwardMessageOpenChunk struct {
	id             string       // chunk ID
	gzipWriter     *gzip.Writer // gzip writer can be nil if no compression, write to reusedGzipBuffer
	numRecords     int          // numbers of collected log records
	numStreamBytes int          // uncompressed length of written stream data
}

// NewMessagePacker creates a LogChunkMaker to pack MessagePackEventStream(s) into Message(s)
// The resulting chunk itself can be saved on disk with ID as the filename, or send as request to upstream
// The maxDataLength is not a hard limit and some of resulted chunks may exceed it
func NewMessagePacker(parentLogger logger.Logger, tag string, mode forwardprotocol.MessageMode) base.LogChunkMaker {
	var asArray bool
	var compressed bool
	switch mode {
	case forwardprotocol.ModeForward:
		asArray = true
		compressed = false
	case forwardprotocol.ModePackedForward:
		asArray = false
		compressed = false
	case forwardprotocol.ModeCompressedPackedForward:
		asArray = false
		compressed = true
	default:
		logger.Panic("unsupported message mode: ", mode)
	}
	msgBuffer := bytes.NewBuffer(make([]byte, 0, messageBufferCapacity))
	return &messagePacker{
		logger:               parentLogger.WithField(defs.LabelComponent, "FluentdForwardMessagePacker"),
		tag:                  tag,
		asArray:              asArray,
		compressed:           compressed,
		reusedGzipBuffer:     bytes.NewBuffer(make([]byte, 0, messageBufferCapacity)),
		reusedMessageBuffer:  msgBuffer,
		reusedMessageEncoder: msgpack.NewEncoder(msgBuffer),
		currentChunk:         nil,
	}
}

func (packer *messagePacker) WriteStream(stream base.LogStream) *base.LogChunk {
	var previousChunk *base.LogChunk
	if packer.currentChunk != nil && packer.currentChunk.numStreamBytes+len(stream) > outputChunkMaxDataBytes {
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
	err := packer.encodeChunkAsMessage(openChunk, packer.reusedGzipBuffer.Bytes())
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
		id:             nextChunkID(),
		gzipWriter:     packer.createGzipWriter(),
		numRecords:     0,
		numStreamBytes: 0,
	}
}

func (packer *messagePacker) createGzipWriter() *gzip.Writer {
	if !compressedMessage {
		return nil
	}
	gzWriter, gzErr := gzip.NewWriterLevel(packer.reusedGzipBuffer, gzipCompressionLevel)
	if gzErr != nil {
		packer.logger.Errorf("failed to create GzipWriter: %s", gzErr.Error())
	}
	return gzWriter
}

func (packer *messagePacker) encodeChunkAsMessage(header forwardMessageOpenChunk, data []byte) error {
	encoder := packer.reusedMessageEncoder

	// root array
	if err := encoder.EncodeArrayLen(3); err != nil {
		return err
	}

	// root[0]: tag
	if err := encoder.EncodeString(packer.tag); err != nil {
		return err
	}

	// root[1]: stream of log events
	if packer.asArray {
		// "Forward" mode: numRecords == the numbers of msgpack objects
		if err := encoder.EncodeArrayLen(header.numRecords); err != nil {
			return err
		}
		if _, err := packer.reusedMessageBuffer.Write(data); err != nil {
			return err
		}
	} else if err := encoder.EncodeBytes(data); err != nil { // "PackedForward" or "CompressedPackedForward" mode
		return err
	}

	// root[2]: option
	option := forwardprotocol.TransportOption{
		Size:       header.numRecords,
		Chunk:      header.id,
		Compressed: "",
	}
	if header.gzipWriter != nil {
		option.Compressed = forwardprotocol.CompressionFormat
	}

	return encoder.Encode(option)
}
