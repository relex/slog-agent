package datadog

import (
	"strings"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
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
	return NewMessagePacker(parentLogger)
}

func (cfg *Config) NewForwarder(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricCreator promreg.MetricCreator) base.ChunkConsumer {
	return NewClientWorker(parentLogger, args, metricCreator, cfg.Upstream)
}

//nolint:revive
func (cfg *Config) VerifyConfig(schema base.LogSchema) error { return nil }
