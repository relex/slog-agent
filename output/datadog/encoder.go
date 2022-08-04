package datadog

import (
	"bytes"
)

type encoder struct {
	reusedMessageBuffer *bytes.Buffer // buffer for final message
}

func newEncoder(reusedMsgBufCapacity int) *encoder {
	return &encoder{
		reusedMessageBuffer: bytes.NewBuffer(make([]byte, 0, reusedMsgBufCapacity)),
	}
}

func (enc *encoder) EncodeChunkAsMessage(data []byte, _ string, _, _ int, _ bool) error {
	_, err := enc.reusedMessageBuffer.Write(data)
	return err
}

func (enc *encoder) GetEncodedResult() []byte {
	return enc.reusedMessageBuffer.Bytes()
}

func (enc *encoder) ResetBuffer() {
	enc.reusedMessageBuffer.Reset()
}
