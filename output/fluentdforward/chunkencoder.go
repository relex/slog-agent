package fluentdforward

import (
	"bytes"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/slog-agent/util"
	"github.com/vmihailenco/msgpack/v4"
)

type encodeChunkParams struct {
	ID           string
	NumRecords   int
	NumBytes     int
	IsCompressed bool
}

type chunkEncoder struct {
	tag                  string
	asArray              bool
	msgpackEncoder       *msgpack.Encoder // encoder for final message
	msgpackEncoderBuffer *bytes.Buffer    // buffer for final message
}

func newEncoder(tag string, asArray bool, msgpackBufferSize int) *chunkEncoder {
	msgpackBuffer := bytes.NewBuffer(make([]byte, 0, msgpackBufferSize))
	return &chunkEncoder{
		tag:                  tag,
		asArray:              asArray,
		msgpackEncoder:       msgpack.NewEncoder(msgpackBuffer),
		msgpackEncoderBuffer: msgpackBuffer,
	}
}

// TODO: The way EncodeChunk works now is to encode last depending on the mode selected,
// but on production we always use the binary mode which is just a copying operation.
// It's a waste of resource and should be changed to be similar to datadog's, though fluentd output is no longer a priority.
func (enc *chunkEncoder) EncodeChunk(data []byte, params *encodeChunkParams) ([]byte, error) {
	defer enc.msgpackEncoderBuffer.Reset()

	// root array
	if err := enc.msgpackEncoder.EncodeArrayLen(3); err != nil {
		return nil, err
	}

	// root[0]: tag
	if err := enc.msgpackEncoder.EncodeString(enc.tag); err != nil {
		return nil, err
	}

	// root[1]: stream of log events
	if enc.asArray {
		// "Forward" mode: numRecords == the numbers of msgpack objects
		if err := enc.msgpackEncoder.EncodeArrayLen(params.NumRecords); err != nil {
			return nil, err
		}
		if _, err := enc.msgpackEncoderBuffer.Write(data); err != nil {
			return nil, err
		}
	} else if err := enc.msgpackEncoder.EncodeBytes(data); err != nil { // "PackedForward" or "CompressedPackedForward" mode
		return nil, err
	}

	// root[2]: option
	option := forwardprotocol.TransportOption{
		Size:       params.NumRecords,
		Chunk:      params.ID,
		Compressed: "",
	}
	if params.IsCompressed {
		option.Compressed = forwardprotocol.CompressionFormat
	}

	if err := enc.msgpackEncoder.Encode(option); err != nil {
		return nil, err
	}

	return util.CopySlice(enc.msgpackEncoderBuffer.Bytes()), nil
}
