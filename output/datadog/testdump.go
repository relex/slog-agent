package datadog

import (
	"bytes"
	"compress/gzip" // DO NOT use klauspost's gzip for verification
	"encoding/json"
	"fmt"
	"io"

	"github.com/relex/slog-agent/base"
)

func dumpDatadogJSON(chunk base.LogChunk, separator []byte, indented bool, writer io.Writer) (base.LogChunkInfo, error) {
	gunzipStream, initErr := gzip.NewReader(bytes.NewReader(chunk.Data)) // use builtin gzip library for verification
	if initErr != nil {
		return base.LogChunkInfo{}, fmt.Errorf("failed to create gzip Reader: %w", initErr)
	}
	inJSON, gzErr := io.ReadAll(gunzipStream)
	if gzErr != nil {
		return base.LogChunkInfo{}, fmt.Errorf("failed to gunzip chunk: %w", gzErr)
	}

	var intermediate []map[string]string
	if decErr := json.Unmarshal(inJSON, &intermediate); decErr != nil {
		return base.LogChunkInfo{}, fmt.Errorf("failed to unmarshal Datadog chunk: %w", decErr)
	}

	if len(intermediate) == 0 {
		return base.LogChunkInfo{
			Tag:        "",
			NumRecords: 0,
		}, nil
	}
	info := base.LogChunkInfo{
		Tag:        intermediate[0]["ddtags"],
		NumRecords: len(intermediate),
	}

	for i, record := range intermediate {
		if i > 0 {
			if _, err := writer.Write(separator); err != nil {
				return info, fmt.Errorf("failed to write separator: %w", err)
			}
		}

		var outJSON []byte
		var outErr error
		if indented {
			outJSON, outErr = json.MarshalIndent(record, "", "  ")
		} else {
			outJSON, outErr = json.Marshal(record)
		}
		if outErr != nil {
			return base.LogChunkInfo{}, fmt.Errorf("failed to marshal decoded Datadog JSON: %w", outErr)
		}

		if _, werr := writer.Write(outJSON); werr != nil {
			return base.LogChunkInfo{}, fmt.Errorf("failed to write decoded Datadog JSON: %w", werr)
		}
	}
	return info, nil
}
