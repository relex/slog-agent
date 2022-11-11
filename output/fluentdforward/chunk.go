package fluentdforward

import (
	"bytes"
	"io"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/output/shared"
	"github.com/relex/slog-agent/util"
)

type encoder interface {
	EncodeChunk(data []byte, params *encodeChunkParams) ([]byte, error)
}

type intermediateChunk struct {
	id                   string
	numRecords, numBytes int
	maxRecords, maxBytes int
	compressor           io.WriteCloser // could be something like a gzip.Writer or nil to disable compression
	writeBuffer          *bytes.Buffer  // an actual buffer that compressor writes to
	encoder              encoder        // a function to finalize/encode the entire chunk after it's been assembled
}

func buildNewChunkFunc(log logger.Logger, initCompressorFunc shared.InitCompessorFunc, encoder encoder, maxRecords, maxBytes int) shared.NewChunkFunc {
	return func(id string, writeBuffer *bytes.Buffer) shared.Chunker {
		chunk := &intermediateChunk{
			id:          id,
			maxRecords:  maxRecords,
			maxBytes:    maxBytes,
			writeBuffer: writeBuffer,
			encoder:     encoder,
		}

		if initCompressorFunc != nil {
			chunk.compressor = initCompressorFunc(log, chunk.writeBuffer)
		}

		return chunk
	}
}

// Write appends new log to log chunk
func (chunk *intermediateChunk) Write(data base.LogStream) error {
	var err error

	if chunk.compressor != nil {
		_, err = chunk.compressor.Write(data)
	} else {
		_, err = chunk.writeBuffer.Write(data)
	}

	if err == nil {
		chunk.numRecords++
		chunk.numBytes += len(data)
	}

	return err
}

func (chunk *intermediateChunk) FinalizeChunk() (*base.LogChunk, error) {
	if chunk.compressor != nil {
		if err := chunk.compressor.Close(); err != nil {
			return nil, err
		}
	}

	defer chunk.writeBuffer.Reset()

	var chunkData []byte

	if chunk.encoder != nil {
		encodeParams := &encodeChunkParams{
			ID:           chunk.id,
			NumRecords:   chunk.numRecords,
			NumBytes:     chunk.numBytes,
			IsCompressed: chunk.compressor != nil,
		}
		encodedChunk, err := chunk.encoder.EncodeChunk(chunk.writeBuffer.Bytes(), encodeParams)
		if err != nil {
			return nil, err
		}
		chunkData = encodedChunk
	} else {
		chunkData = util.CopySlice(chunk.writeBuffer.Bytes())
	}

	return &base.LogChunk{
		ID:    chunk.id,
		Data:  chunkData,
		Saved: false,
	}, nil
}

func (chunk *intermediateChunk) CanAppendData(dataLength int) bool {
	// flush when the amount of log records reaches max permitted amount, if it is defined
	if chunk.maxRecords > 0 && chunk.numRecords >= chunk.maxRecords {
		return false
	}
	// otherwise flush when the total size reaches max permitted amount
	if chunk.maxBytes > 0 && chunk.numBytes+dataLength > chunk.maxBytes {
		return false
	}

	return true
}
