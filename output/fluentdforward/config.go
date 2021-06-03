package fluentdforward

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
)

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
	Address string `yaml:"address"`
	TLS     bool   `yaml:"tls"`
	Secret  string `yaml:"secret"`
}

// DumpRecordsAsJSON decodes and dumps log records in chunk as JSON format
func (cfg *Config) DumpRecordsAsJSON(chunk base.LogChunk, separator []byte, indented bool, destination io.Writer) (base.LogChunkInfo, error) {
	return decodeAndDumpRecordsAsJSON(chunk, separator, indented, destination)
}

// MatchChunkID checks whether given ID is valid for a fluentdforward chunk file
func (cfg *Config) MatchChunkID(chunkID string) bool {
	return strings.HasSuffix(chunkID, chunkIDSuffix)
}

// NewSerializer creates LogSerializer
func (cfg *Config) NewSerializer(parentLogger logger.Logger, schema base.LogSchema, deallocator *base.LogAllocator) base.LogSerializer {
	return MustNewEventSerializer(parentLogger, schema, cfg.Serialization, deallocator)
}

// NewChunkMaker creates LogChunkMaker
func (cfg *Config) NewChunkMaker(parentLogger logger.Logger, tag string) base.LogChunkMaker {
	return NewMessagePacker(parentLogger, tag, cfg.MessageMode)
}

// NewForwarder creates the forwarding client
func (cfg *Config) NewForwarder(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricFactory *base.MetricFactory) base.ChunkConsumer {
	return NewClientWorker(parentLogger, args, cfg.Upstream, metricFactory)
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
		break
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
	return nil
}
