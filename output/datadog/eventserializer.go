package datadog

import (
	"encoding/json"
	"strconv"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

type eventSerializer struct {
	logger logger.Logger
	config SerializationConfig
	schema base.LogSchema
}

func NewEventSerializer(parentLogger logger.Logger, schema base.LogSchema, config SerializationConfig) base.LogSerializer {
	return &eventSerializer{
		logger: parentLogger,
		schema: schema,
		config: config,
	}
}

func (packer *eventSerializer) SerializeRecord(record *base.LogRecord) base.LogStream {
	fieldNames := packer.schema.GetFieldNames()
	outputMap := make(map[string]*string, len(fieldNames)+4)

	for i, fieldName := range fieldNames {
		if i > len(record.Fields) {
			break
		}
		if fieldName != "" && record.Fields[i] != "" {
			outputMap[fieldName] = &record.Fields[i]
		}
	}

	outputMap["ddsource"] = packer.config.Source
	outputMap["ddtags"] = packer.config.Tags
	outputMap["service"] = packer.config.Service

	timestamp := strconv.FormatInt(record.Timestamp.UnixMilli(), 10)
	outputMap["timestamp"] = &timestamp

	// TODO: might consider using a faster marshaller such as go-json
	data, _ := json.Marshal(outputMap) //nolint:errcheck // errors can't pop up here
	return data
}
