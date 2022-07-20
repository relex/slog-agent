package hybridbuffer

import (
	"fmt"
	"os"
	"strings"

	"github.com/c2h5oh/datasize"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/defs"
)

// Config defines the configuration for HybridBufferer
type Config struct {
	bconfig.Header `yaml:",inline"`
	RootPath       string            `yaml:"rootPath"`   // root path on top of buffer subdirs, may contain environment variables
	MaxBufSize     datasize.ByteSize `yaml:"maxBufSize"` // max total size of on-disk chunks for each buffer subdir
}

// ListBufferIDs lists existing buffer IDs
func (cfg *Config) ListBufferIDs(parentLogger logger.Logger, matchChunkID func(string) bool,
	metricCreator promreg.MetricCreator) []string {

	clogger := parentLogger.WithField(defs.LabelComponent, "HybridBufferConfig")

	rootPath := os.ExpandEnv(cfg.RootPath)
	return listBufferQueueIDs(clogger, rootPath, matchChunkID, metricCreator)
}

// NewBufferer creates a HybridBufferer
// If bufferID is empty, the queue dir is the root dir as defined in .path
func (cfg *Config) NewBufferer(parentLogger logger.Logger, bufferID string, matchChunkID func(string) bool,
	metricCreator promreg.MetricCreator, sendAllAtEnd bool) base.ChunkBufferer {

	rootPath := os.ExpandEnv(cfg.RootPath)
	if strings.Contains(rootPath, "$") {
		parentLogger.Warnf("possibly misconfigured .rootPath: '%s'", rootPath)
	}

	return newBufferer(parentLogger, rootPath, bufferID, matchChunkID, metricCreator, int64(cfg.MaxBufSize.Bytes()), sendAllAtEnd)
}

// VerifyConfig checks configuration
func (cfg *Config) VerifyConfig() error {
	if len(cfg.RootPath) == 0 {
		return fmt.Errorf(".rootPath is unspecified")
	}
	if cfg.MaxBufSize.Bytes() == 0 {
		return fmt.Errorf(".maxBufSize is unspecified")
	}
	return nil
}
