package datadog

import (
	"encoding/json"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/output/shared"
	"github.com/stretchr/testify/assert"
)

func getStringPtr(data string) *string {
	return &data
}

var testCfg = SerializationConfig{
	Source:  getStringPtr("testSource"),
	Tags:    getStringPtr("testTags"),
	Service: getStringPtr("testService"),
}

func TestSerializer_Succeeds(t *testing.T) {
	serializer := NewEventSerializer(logger.Root(), shared.TestSchema, testCfg)
	for _, record := range shared.TestInputRecords {
		stream := serializer.SerializeRecord(shared.TestSchema.CopyTestRecord(record))
		verifyLogStream(t, stream, record)
	}
}

func verifyLogStream(t *testing.T, stream base.LogStream, record *base.LogRecord) {
	var streamData map[string]string
	assert.NoError(t, json.Unmarshal(stream, &streamData))

	assert.Equal(t, streamData["ddsource"], *testCfg.Source)
	assert.Equal(t, streamData["ddtags"], *testCfg.Tags)
	assert.Equal(t, streamData["service"], *testCfg.Service)
	assert.Equal(t, streamData["timestamp"], record.Timestamp.String())

	schemaFields := shared.TestSchema.GetFieldNames()
	assert.Equal(t, len(record.Fields), len(schemaFields))
	for i, fieldName := range schemaFields {
		expectedValue := record.Fields[i]
		actualData := streamData[fieldName]
		assert.Equal(t, expectedValue, actualData)
	}
}
