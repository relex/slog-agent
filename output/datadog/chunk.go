package datadog

import (
	"bytes"
	"io"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/output/shared"
	"github.com/relex/slog-agent/util"
)

type intermediateChunk struct {
	id                   string
	numRecords, numBytes int
	maxRecords, maxBytes int
	compressor           io.WriteCloser // could be something like a gzip.Writer or nil to disable compression
	writeBuffer          *bytes.Buffer  // an actual buffer that compressor writes to
}

func buildNewChunkFunc(log logger.Logger, maxRecords, maxBytes int) shared.NewChunkFunc {
	return func(id string, writeBuffer *bytes.Buffer) shared.Chunker {
		chunk := &intermediateChunk{
			id:          id,
			maxRecords:  maxRecords,
			maxBytes:    maxBytes,
			compressor:  nil,
			writeBuffer: writeBuffer,
		}

		chunk.compressor = shared.InitGzipCompessor(log, chunk.writeBuffer)

		_, err := chunk.compressor.Write([]byte("["))
		if err != nil {
			log.Error(err)
		} else {
			chunk.numBytes += len("[")
		}

		return chunk
	}
}

const recordDelimiter = (",")

// Write appends new log to log chunk
func (chunk *intermediateChunk) Write(data base.LogStream) error {
	var err error

	var writer io.Writer = chunk.writeBuffer
	if chunk.compressor != nil {
		writer = chunk.compressor
	}

	// in order to build a correct json array [{},{},{}] - we need to append commas before every record other than the first one
	if chunk.numRecords != 0 {
		_, err = writer.Write(base.LogStream(recordDelimiter))
		if err != nil {
			return err
		}
	}

	_, err = writer.Write(data)
	if err == nil {
		chunk.numRecords++
		chunk.numBytes += len(recordDelimiter) + len(data)
	}

	return err
}

func (chunk *intermediateChunk) FinalizeChunk() (*base.LogChunk, error) {
	if chunk.compressor != nil {
		if _, err := chunk.compressor.Write([]byte("]")); err != nil {
			return nil, err
		}

		if err := chunk.compressor.Close(); err != nil {
			return nil, err
		}
	} else {
		if _, err := chunk.writeBuffer.Write([]byte("]")); err != nil {
			return nil, err
		}
	}
	chunk.numBytes += len("]")
	defer chunk.writeBuffer.Reset()

	return &base.LogChunk{
		ID:    chunk.id,
		Data:  util.CopySlice(chunk.writeBuffer.Bytes()),
		Saved: false,
	}, nil
}

func (chunk *intermediateChunk) CanAppendData(dataLength int) bool {
	// flush when the amount of log records reaches max permitted amount, if it is defined
	if chunk.maxRecords > 0 && chunk.numRecords >= chunk.maxRecords {
		return false
	}
	// otherwise flush when the total size reaches max permitted amount
	if chunk.maxBytes > 0 && chunk.numBytes+dataLength+len("]") > chunk.maxBytes {
		return false
	}

	return true
}
