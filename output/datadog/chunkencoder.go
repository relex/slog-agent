package datadog

import (
	"github.com/relex/slog-agent/output/shared"
	"github.com/relex/slog-agent/util"
)

type chunkEncoder struct{}

func newEncoder() *chunkEncoder {
	return &chunkEncoder{}
}

//nolint:revive
func (enc *chunkEncoder) EncodeChunk(chunk *shared.BasicChunk) ([]byte, error) {
	return util.CopySlice(chunk.Bytes()), nil
}
