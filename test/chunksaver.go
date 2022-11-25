package test

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

// chunkSaver collects processed logs for pipeline-based benchmarks and integration tests
type chunkSaver interface {
	Write(chunk base.LogChunk, decoder base.ChunkDecoder)
	Close()
}

func newChunkSaver(outputName string, outputPathPattern string) chunkSaver {
	expandedOutputPath := expandPathWithOutputName(outputPathPattern, outputName)

	switch {
	case outputPathPattern == "null":
		logger.Infof("open chunk null output")
		return &nullChunkSaver{}

	case strings.Contains(expandedOutputPath, "%s"):
		logger.Infof("open per-chunk JSON file output %s", expandedOutputPath)
		return &perChunkChunkSaver{
			outputName:       outputName,
			outputPathFormat: expandedOutputPath,
		}

	default:
		logger.Infof("open single JSON file output %s", expandedOutputPath)
		return newSingleFileChunkSaver(expandedOutputPath)
	}
}

func expandPathWithOutputName(pattern string, outputName string) string {
	return os.Expand(pattern, func(varName string) string {
		switch varName {
		case "OUTPUT":
			return outputName
		default:
			return os.Getenv(varName)
		}
	})
}

type nullChunkSaver struct{}

func (s *nullChunkSaver) Write(chunk base.LogChunk, decoder base.ChunkDecoder) {}

func (s *nullChunkSaver) Close() {}

type perChunkChunkSaver struct {
	outputName       string
	outputPathFormat string
}

func (s *perChunkChunkSaver) Write(chunk base.LogChunk, decoder base.ChunkDecoder) {
	chunkFilePath := fmt.Sprintf(s.outputPathFormat, chunk.ID) // replace "%s" with chunk id
	tmp := &bytes.Buffer{}
	tmp.WriteString("[\n")
	if _, err := decoder.DecodeChunkToJSON(chunk, []byte(",\n"), true, tmp); err != nil {
		logger.Panicf("error dumping records: %s", err.Error())
	}
	tmp.WriteString("\n]\n")
	if err := os.WriteFile(chunkFilePath, tmp.Bytes(), 0644); err != nil {
		logger.Panicf("error writing to %s: %s", chunkFilePath, err.Error())
	}
}

func (s *perChunkChunkSaver) Close() {}

type singleFileChunkSaver struct {
	outputFile   *os.File
	outputBuffer *bufio.Writer
}

func newSingleFileChunkSaver(outputPath string) *singleFileChunkSaver {
	file, ferr := os.Create(outputPath)
	if ferr != nil {
		logger.Panicf("error creating %s: %s", outputPath, ferr.Error())
	}
	buffer := bufio.NewWriterSize(file, 1048576)
	if _, err := buffer.WriteString("[\n"); err != nil {
		logger.Panicf("error writing to %s: %s", outputPath, ferr.Error())
	}
	return &singleFileChunkSaver{
		outputFile:   file,
		outputBuffer: buffer,
	}
}

func (s *singleFileChunkSaver) Write(chunk base.LogChunk, decoder base.ChunkDecoder) {
	_, err := decoder.DecodeChunkToJSON(chunk, []byte(",\n"), true, s.outputBuffer)
	if err != nil {
		logger.Panicf("error writing to %s: %s", s.outputFile.Name(), err.Error())
	}
}

func (s *singleFileChunkSaver) Close() {
	if _, err := s.outputBuffer.WriteString("\n]\n"); err != nil {
		logger.Panicf("error writing JSON end to %s: %s", s.outputFile.Name(), err.Error())
	}
	if err := s.outputBuffer.Flush(); err != nil {
		logger.Panicf("error flushing %s: %s", s.outputFile.Name(), err.Error())
	}
	if err := s.outputFile.Close(); err != nil {
		logger.Panicf("error closing %s: %s", s.outputFile.Name(), err.Error())
	}
}
