package obykeyset

//go:generate genny -in=../templatepkg/globalmap.tpl.go -out=globalmap.gen.go gen "templatepkg=obykeyset BaseName=PipelineChannel globalObjectType=pipelineChannel localWrapperType=*pipelineChannelLocalBuffer"

import (
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/orchestrate/obase"
)

type pipelineChannel = obase.PipelineChannel
type pipelineChannelLocalBuffer = obase.PipelineChannelLocalBuffer

func closePipelineChannel(ch chan<- []*base.LogRecord) {
	close(ch)
}
