package bconfig

import (
	"io"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
)

// LogOutput is split into
//   1. LogSerializer: serialize log records one by one
//   2. LogChunkMaker: aggregate serialized records into chunks ready for persistance or transport
//   3. ChunkConsumer: save or forward chunks to somewhere
//
// A ChunkBufferer is inserted between one or more LogChunkMaker(s) and one ChunkConsumer to support e.g. on-disk buffering

// LogOutputConfig provides an interface for the configuration of LogSerializer, LogChunkMaker and ChunkConsumer
//
// All the implementations should support YAML unmarshalling
type LogOutputConfig interface {
	BaseConfig

	DumpRecordsAsJSON(chunk base.LogChunk, separator []byte, indented bool, destination io.Writer) (base.LogChunkInfo, error)

	MatchChunkID(chunkID string) bool

	NewSerializer(parentLogger logger.Logger, schema base.LogSchema, deallocator *base.LogAllocator) base.LogSerializer

	NewChunkMaker(parentLogger logger.Logger, tag string) base.LogChunkMaker

	NewForwarder(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricCreator promreg.MetricCreator) base.ChunkConsumer

	VerifyConfig(schema base.LogSchema) error
}

type LogOutputConfigHolder = ConfigHolder[LogOutputConfig]
type LogOutputConfigCreatorTable = ConfigCreatorTable[LogOutputConfig]
