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

// convertMsgpackToJSON converts msgpack (fluentd protocol) to json
func convertMsgpackToJSON(chunk base.LogChunk, separator []byte, indented bool, writer io.Writer) (base.LogChunkInfo, error) {
	reader := bytes.NewReader(chunk.Data)
	var message forwardprotocol.Message
	if merr := msgpack.NewDecoder(reader).Decode(&message); merr != nil {
		return base.LogChunkInfo{}, fmt.Errorf("failed to decode chunk: %w", merr)
	}

	info := base.LogChunkInfo{
		Tag:        message.Tag,
		NumRecords: len(message.Entries),
	}

	for i, log := range message.Entries {
		if i > 0 {
			if _, err := writer.Write(separator); err != nil {
				return info, fmt.Errorf("failed to write separator: %w", err)
			}
		}
		jbin, jerr := dump.FormatEventInJSON(log, message.Tag, indented)
		if jerr != nil {
			return info, fmt.Errorf("failed to format log in JSON: %w", jerr)
		}
		if _, err := writer.Write(jbin); err != nil {
			return info, fmt.Errorf("failed to write log: %w", err)
		}
	}
	return info, nil
}
