package fluentdforward

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/output/shared"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v4"
)

var testSerializationConfig = SerializationConfig{
	EnvironmentFields: []string{"vhost", "app"},
	HiddenFields:      []string{"comp"},
	RewriteFields: map[string][]bconfig.LogRewriterConfigHolder{
		"message": {
			{Location: "", Value: &shared.TestLogRewriter1{}},
			{Location: "", Value: &shared.TestLogRewriter2{}},
		},
	},
}

func TestForwardLogEventSerializer(t *testing.T) {
	serializer, err := NewEventSerializer(logger.Root(), shared.TestSchema, testSerializationConfig)
	assert.Nil(t, err)
	for i, record := range shared.TestInputRecords {
		stream := serializer.SerializeRecord(shared.TestSchema.CopyTestRecord(record))
		decoder := msgpack.NewDecoder(bytes.NewBuffer(stream))
		testVerifyLogRecord(t, i, decoder)
		_, err = decoder.DecodeInterface()
		assert.EqualError(t, err, "EOF", i)
	}
}

func testVerifyLogRecord(t *testing.T, nth int, decoder *msgpack.Decoder) {
	var entry forwardprotocol.EventEntry
	assert.Nil(t, decoder.Decode(&entry), fmt.Sprintf("record[%d] err", nth))
	assert.Equal(t, shared.TestInputRecords[nth].Timestamp.UnixNano(), entry.Time.UnixNano(), fmt.Sprintf("record[%d] timestamp", nth))
	assert.Equal(t, shared.TestOutputFieldMaps[nth], entry.Record, fmt.Sprintf("record[%d] fields", nth))
}
