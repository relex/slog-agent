package fluentdforward

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/output/shared"
)

// output-specific file extension for generated chunks
const chunkIDSuffix = ".ff"

// msgBufCapacity is the initial capacity for buffers used for chunk and compression
// It only needs to be large enough to contain the largest uncompressed message
const msgBufCapacity = 1 * 1024 * 1024

// chunkMaxSizeBytes defines the max uncompressed data size of a LogChunk, not including necessary headers.
// The value must be well below Fluentd's Fluent::Plugin::Buffer::DEFAULT_CHUNK_LIMIT_SIZE, as some buffers are implicitly inserted and non-configurable.
//
// https://github.com/fluent/fluentd/blob/master/lib/fluent/plugin/buffer.rb#L39
var chunkMaxSizeBytes = 7 * 1024 * 1024

// chunkMaxRecords is the max amount of log entries a chunk can hold before flushing.
// Can be 0 in case there's no limit.
var chunkMaxRecords = 0

// Config defines configuration for fluentd-forward output
type Config struct {
	bconfig.Header `yaml:",inline"`
	Serialization  SerializationConfig         `yaml:"serialization"`
	MessageMode    forwardprotocol.MessageMode `yaml:"messageMode"`
	Upstream       UpstreamConfig              `yaml:"upstream"`
}

// SerializationConfig defines the serialization section in config file
type SerializationConfig struct {
	EnvironmentFields []string                                     `yaml:"environmentFields"`
	HiddenFields      []string                                     `yaml:"hiddenFields"`
	RewriteFields     map[string][]bconfig.LogRewriterConfigHolder `yaml:"rewriteFields"`
}

// UpstreamConfig defines the upstream section in config file
type UpstreamConfig struct {
	Address     string        `yaml:"address"`
	TLS         bool          `yaml:"tls"`
	Secret      string        `yaml:"secret"`
	MaxDuration time.Duration `yaml:"maxDuration"`
}

// MatchChunkID checks whether given ID is valid for a fluentdforward chunk file
func (cfg *Config) MatchChunkID(chunkID string) bool { //nolint:revive
	return strings.HasSuffix(chunkID, chunkIDSuffix)
}

// NewSerializer creates LogSerializer
func (cfg *Config) NewSerializer(parentLogger logger.Logger, schema base.LogSchema) base.LogSerializer {
	return MustNewEventSerializer(parentLogger, schema, cfg.Serialization)
}

// NewChunkMaker creates LogChunkMaker
func (cfg *Config) NewChunkMaker(parentLogger logger.Logger, tag string) base.LogChunkMaker {
	var asArray bool

	var initCompressorFunc shared.InitCompressorFunc
	switch cfg.MessageMode {
	case forwardprotocol.ModeForward:
		asArray = true
	case forwardprotocol.ModePackedForward:
	case forwardprotocol.ModeCompressedPackedForward:
		initCompressorFunc = shared.InitGzipCompessor
	default:
		parentLogger.Fatalf("unsupported message mode: %s", cfg.MessageMode)
	}

	encoder := newEncoder(tag, asArray, msgBufCapacity)
	newChunkFunc := buildNewChunkFunc(parentLogger, initCompressorFunc, encoder, chunkMaxRecords, chunkMaxSizeBytes)
	chunkFactory := shared.NewChunkFactory(chunkIDSuffix, msgBufCapacity, newChunkFunc)

	return shared.NewMessagePacker(parentLogger, chunkFactory)
}

// NewForwarder creates the forwarding client
func (cfg *Config) NewForwarder(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricCreator promreg.MetricCreator) base.ChunkConsumer {
	return NewClientWorker(parentLogger, args, cfg.Upstream, metricCreator)
}

// VerifyConfig verifies the configuration
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if len(cfg.Serialization.EnvironmentFields) == 0 {
		return fmt.Errorf(".serialization.environmentFields is unspecified")
	}

	for field, rewriteConfig := range cfg.Serialization.RewriteFields {
		if _, err := schema.CreateFieldLocator(field); err != nil {
			return fmt.Errorf(".serialization.rewriteFields[%s]: Field is invalid: %w", field, err)
		}
		if err := bsupport.VerifyRewriterConfigs(rewriteConfig, schema, fmt.Sprintf(".serialization.rewriteFields[%s]", field)); err != nil {
			return err
		}
	}

	switch cfg.MessageMode {
	case "":
		return fmt.Errorf(".messageMode is unspecified")
	case forwardprotocol.ModeForward:
	case forwardprotocol.ModePackedForward:
	case forwardprotocol.ModeCompressedPackedForward:
	default:
		return fmt.Errorf(".messageMode: '%s' is not a valid mode", cfg.MessageMode)
	}

	if len(cfg.Upstream.Address) == 0 {
		return fmt.Errorf(".upstream.address is unspecified")
	}
	if _, _, err := net.SplitHostPort(cfg.Upstream.Address); err != nil {
		return fmt.Errorf(".upstream.address is invalid: %w", err)
	}

	if cfg.Upstream.TLS && len(cfg.Upstream.Secret) == 0 {
		return fmt.Errorf(".upstream.secret is unspecified when tls=true")
	}

	if cfg.Upstream.MaxDuration == 0 {
		return fmt.Errorf(".upstream.maxDuration is unspecified")
	}
	return nil
}
