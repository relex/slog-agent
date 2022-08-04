package fluentdforward

import (
	"bytes"
	"fmt"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/vmihailenco/msgpack/v4"
)

type encoder struct {
	tag                  string
	asArray              bool
	reusedMsgpackEncoder *msgpack.Encoder // encoder for final message
	reusedMessageBuffer  *bytes.Buffer    // buffer for final message
}

func newEncoder(tag string, mode forwardprotocol.MessageMode, reusedMsgBufCapacity int) (*encoder, error) {
	var asArray bool

	switch mode {
	case forwardprotocol.ModeForward:
		asArray = true
	case forwardprotocol.ModePackedForward:
		asArray = false
	case forwardprotocol.ModeCompressedPackedForward:
		asArray = false
	default:
		return nil, fmt.Errorf("unsupported message mode: %s", mode)
	}

	msgBuffer := bytes.NewBuffer(make([]byte, 0, reusedMsgBufCapacity))

	return &encoder{
		tag:                  tag,
		asArray:              asArray,
		reusedMsgpackEncoder: msgpack.NewEncoder(msgBuffer),
		reusedMessageBuffer:  msgBuffer,
	}, nil
}

func (enc *encoder) EncodeChunkAsMessage(data []byte, id string, numRecords, _ int, isCompressed bool) error {
	encoder := enc.reusedMsgpackEncoder

	// root array
	if err := encoder.EncodeArrayLen(3); err != nil {
		return err
	}

	// root[0]: tag
	if err := encoder.EncodeString(enc.tag); err != nil {
		return err
	}

	// root[1]: stream of log events
	if enc.asArray {
		// "Forward" mode: numRecords == the numbers of msgpack objects
		if err := encoder.EncodeArrayLen(numRecords); err != nil {
			return err
		}
		if _, err := enc.reusedMessageBuffer.Write(data); err != nil {
			return err
		}
	} else if err := encoder.EncodeBytes(data); err != nil { // "PackedForward" or "CompressedPackedForward" mode
		return err
	}

	// root[2]: option
	option := forwardprotocol.TransportOption{
		Size:       numRecords,
		Chunk:      id,
		Compressed: "",
	}
	if isCompressed {
		option.Compressed = forwardprotocol.CompressionFormat
	}

	return encoder.Encode(option)
}

func (enc *encoder) GetEncodedResult() []byte {
	return enc.reusedMessageBuffer.Bytes()
}

func (enc *encoder) ResetBuffer() {
	enc.reusedMessageBuffer.Reset()
}
