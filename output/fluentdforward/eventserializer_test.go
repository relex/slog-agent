package fluentdforward

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/gotils/logger"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v4"
)

func TestForwardLogEventSerializer(t *testing.T) {
	serializer, err := NewEventSerializer(logger.Root(), testSchema, testSerializationConfig)
	assert.Nil(t, err)
	for i, record := range testInputRecords {
		stream := serializer.SerializeRecord(testSchema.CopyTestRecord(record))
		decoder := msgpack.NewDecoder(bytes.NewBuffer(stream))
		testVerifyLogRecord(t, i, decoder)
		_, err = decoder.DecodeInterface()
		assert.EqualError(t, err, "EOF", i)
	}
}

func testVerifyLogRecord(t *testing.T, nth int, decoder *msgpack.Decoder) {
	var entry forwardprotocol.EventEntry
	assert.Nil(t, decoder.Decode(&entry), fmt.Sprintf("record[%d] err", nth))
	assert.Equal(t, testInputRecords[nth].Timestamp.UnixNano(), entry.Time.UnixNano(), fmt.Sprintf("record[%d] timestamp", nth))
	assert.Equal(t, testOutputFieldMaps[nth], entry.Record, fmt.Sprintf("record[%d] fields", nth))
}
