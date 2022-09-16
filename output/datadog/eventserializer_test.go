package datadog

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/output/shared"
	"github.com/stretchr/testify/assert"
)

var testCfg = SerializationConfig{
	HiddenFields: []string{},
}

func TestSerializer_Succeeds(t *testing.T) {
	serializer := NewEventSerializer(logger.Root(), shared.TestSchema, testCfg, "testTag")
	for _, record := range shared.TestInputRecords {
		stream := serializer.SerializeRecord(shared.TestSchema.CopyTestRecord(record))
		verifyLogStream(t, stream, record)
	}
}

func verifyLogStream(t *testing.T, stream base.LogStream, record *base.LogRecord) {
	var streamData map[string]string
	assert.NoError(t, json.Unmarshal(stream, &streamData))

	assert.Equal(t, streamData["timestamp"], strconv.FormatInt(record.Timestamp.UnixMilli(), 10))
	assert.Equal(t, streamData["ddtags"], "testTag")

	schemaFields := shared.TestSchema.GetFieldNames()
	assert.Equal(t, len(record.Fields), len(schemaFields))
	for i, fieldName := range schemaFields {
		expectedValue := record.Fields[i]
		actualData := streamData[fieldName]
		assert.Equal(t, expectedValue, actualData)
	}
}
