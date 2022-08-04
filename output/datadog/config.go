package datadog

import (
	"strings"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/output/shared"
)

const (
	// chunkIDSuffix is an output-specific file extension for generated chunks.
	chunkIDSuffix = ".dd"

	// chunkMaxSizeBytes defines the max uncompressed data size of a LogChunk.
	// See the api docs here: https://docs.datadoghq.com/api/latest/logs/#send-logs
	chunkMaxSizeBytes = 5 * 1024 * 1024

	// chunkMaxRecords is the max amount of log entries a chunk can hold before flushing.
	// Can be 0 in case there's no limit.
	chunkMaxRecords = 1000

	// messageBufferCapacity is the initial capacity for buffers used for chunk and compression.
	// It only needs to be large enough to contain the largest compressed message.
	msgBufCapacity = 1 * 1024 * 1024
)

type Config struct {
	bconfig.Header `yaml:",inline"`
	Serialization  SerializationConfig
	Upstream       UpstreamConfig
}

type SerializationConfig struct {
	Source  *string
	Tags    *string
	Service *string
}

type UpstreamConfig struct {
	Address     string        `yaml:"address"`
	APIKey      string        `yaml:"apiKey"`
	HTTPTimeout time.Duration `yaml:"httpTimeout"`
}

//nolint:revive
func (cfg *Config) MatchChunkID(chunkID string) bool {
	return strings.HasSuffix(chunkID, chunkIDSuffix)
}

func (cfg *Config) NewSerializer(parentLogger logger.Logger, schema base.LogSchema) base.LogSerializer {
	return NewEventSerializer(parentLogger, schema, cfg.Serialization)
}

//nolint:revive
func (cfg *Config) NewChunkMaker(parentLogger logger.Logger, tag string) base.LogChunkMaker {
	packerCfg := &shared.PackerConfig{
		MsgBufCapacity:    msgBufCapacity,
		ChunkMaxSizeBytes: chunkMaxSizeBytes,
		ChunkMaxRecords:   chunkMaxRecords,
		ChunkIDSuffix:     chunkIDSuffix,
		UseCompression:    true,
	}

	return shared.NewMessagePacker(parentLogger, packerCfg, newEncoder(msgBufCapacity))
}

func (cfg *Config) NewForwarder(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricCreator promreg.MetricCreator) base.ChunkConsumer {
	return NewClientWorker(parentLogger, args, metricCreator, cfg.Upstream)
}

//nolint:revive
func (cfg *Config) VerifyConfig(schema base.LogSchema) error { return nil }
