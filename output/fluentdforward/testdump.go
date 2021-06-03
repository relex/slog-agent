package fluentdforward

import (
	"bytes"
	"fmt"
	"io"

	"github.com/relex/fluentlib/dump"
	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/slog-agent/base"
	"github.com/vmihailenco/msgpack/v4"
)

func decodeAndDumpRecordsAsJSON(chunk base.LogChunk, separator []byte, indented bool, writer io.Writer) (base.LogChunkInfo, error) {
	reader := bytes.NewReader(chunk.Data)
	var message forwardprotocol.Message
	if merr := msgpack.NewDecoder(reader).Decode(&message); merr != nil {
		return base.LogChunkInfo{}, fmt.Errorf("failed to decode chunk: %w", merr)
	}
	info := base.LogChunkInfo{
		Tag:        message.Tag,
		NumRecords: len(message.Entries),
	}
	return info, dump.PrintFromMessageToJSON(message, separator, indented, writer)
}
