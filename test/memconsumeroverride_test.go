package test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

type outputBufferByTag map[string]*bytes.Buffer
type multiOutputCollector map[string]outputBufferByTag // output => tag => JSON buffer

// prepareInMemoryConsumerOverride creates in-memory per-tag output collectors (with workers) to be used with worker/agent-based integration tests
func prepareInMemoryConsumerOverride(t *testing.T) (multiOutputCollector, base.ChunkConsumerOverrideCreator) {
	outputChunkSavers := make(map[string]*sharedByTagChunkSaver) // output name => chunk saver (one saver for all pipelines)
	outputCollector := make(multiOutputCollector)
	mapLock := &sync.Mutex{}

	return outputCollector,
		func(parentLogger logger.Logger, outputName string, decoder base.ChunkDecoder, args base.ChunkConsumerArgs) base.ChunkConsumer {
			mapLock.Lock()
			defer mapLock.Unlock()

			saver, exists := outputChunkSavers[outputName]
			if !exists {
				saver = newSharedByTagChunkSaver(t, outputName)
				outputChunkSavers[outputName] = saver
				outputCollector[outputName] = saver.outputBuffers
			}
			return newChunkSavingWorker(parentLogger.WithField("output", outputName), decoder, args, saver)
		}
}

type sharedByTagChunkSaver struct {
	logger        logger.Logger
	outputBuffers outputBufferByTag
	outputLock    *sync.Mutex
	closed        bool
}

func newSharedByTagChunkSaver(t *testing.T, outputName string) *sharedByTagChunkSaver {
	return &sharedByTagChunkSaver{
		logger: logger.WithFields(logger.Fields{
			"test":   t.Name(),
			"output": outputName,
		}),
		outputBuffers: make(outputBufferByTag),
		outputLock:    &sync.Mutex{},
		closed:        false,
	}
}

func (s *sharedByTagChunkSaver) Write(chunk base.LogChunk, decoder base.ChunkDecoder) {
	s.outputLock.Lock()
	defer s.outputLock.Unlock()

	buf := bytes.Buffer{}
	info, derr := decoder.DecodeChunkToJSON(chunk, []byte(",\n"), true, &buf)
	if derr != nil {
		s.logger.Errorf("failed to decode chunk: %s", derr.Error())
		return
	}
	s.logger.Infof("chunk: tag=%s count=%d", info.Tag, info.NumRecords)

	wrt := s.getOutputBuffer(info.Tag)
	if wrt.Len() == 0 {
		if _, err := wrt.Write([]byte("[\n")); err != nil {
			s.logger.Errorf("failed to write separator: %v", err)
			return
		}
	} else {
		if _, err := wrt.Write([]byte(",\n")); err != nil {
			s.logger.Errorf("failed to write separator: %v", err)
			return
		}
	}
	if _, werr := wrt.Write(buf.Bytes()); werr != nil {
		s.logger.Errorf("failed to write JSON: %s: %v", chunk.ID, werr)
		return
	}

}

func (s *sharedByTagChunkSaver) Close() {
	s.outputLock.Lock()
	defer s.outputLock.Unlock()

	if s.closed {
		return
	}

	for tag, buf := range s.outputBuffers {
		if _, err := buf.Write([]byte("\n]\n")); err != nil {
			s.logger.Fatalf("failed to write end bracket for tag=%s: %s", tag, err.Error())
		}
	}
	s.closed = true
}

func (s *sharedByTagChunkSaver) getOutputBuffer(tag string) *bytes.Buffer {
	buf, exists := s.outputBuffers[tag]
	if !exists {
		buf = bytes.NewBuffer(make([]byte, 0, 1048576))
		s.outputBuffers[tag] = buf
	}
	return buf
}
