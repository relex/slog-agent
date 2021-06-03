// Package buffer registers the list of all ChunkBufferer implementations
package buffer

import (
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/buffer/hybridbuffer"
)

func init() {
	bconfig.RegisterChunkBufferConfigConstructors(map[string]func() bconfig.ChunkBufferConfig{
		"hybridBuffer": func() bconfig.ChunkBufferConfig { return &hybridbuffer.Config{} },
	})
}

// Register registers all input config types
func Register() {
	// trigger init()
}
