package bconfig

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
)

// ChunkBufferConfig provides an interface for the configuration of ChunkBufferer(s)
// All the implementations should support YAML unmarshalling
type ChunkBufferConfig interface {

	// ListBufferIDs lists existing buffer IDs for orchestrator(s) to re-create their pipelines and start recovery
	ListBufferIDs(parentLogger logger.Logger, matchChunkID func(string) bool,
		metricCreator promreg.MetricCreator) []string

	// NewBufferer creates a ChunkBufferer
	// bufferID: global and unique ID within the config; may point to an existing storage (e.g. name of queue dir)
	// matchChunkID: function to match chunkID in existing storage, in case of protocol change in chunk-making
	// sendAllAtEnd: for test, make sure all chunks are sent and consumed at shutdown instead of saving them
	NewBufferer(parentLogger logger.Logger, bufferID string, matchChunkID func(string) bool,
		metricCreator promreg.MetricCreator, sendAllAtEnd bool) base.ChunkBufferer

	// VerifyConfig verifies the configuration
	VerifyConfig() error
}
