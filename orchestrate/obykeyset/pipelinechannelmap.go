package obykeyset

import (
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/orchestrate/obase"
	"github.com/relex/slog-agent/util/localcachedmap"
)

type globalPipelineChannelMap = localcachedmap.GlobalCachedMap[obase.PipelineChannel, *obase.PipelineChannelLocalBuffer]

type localPipelineChannelMap = localcachedmap.LocalCachedMap[obase.PipelineChannel, *obase.PipelineChannelLocalBuffer]

func closePipelineChannel(ch chan<- []*base.LogRecord) {
	close(ch)
}
