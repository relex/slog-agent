package datadog

import (
	"encoding/json"

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

	timestamp := record.Timestamp.String()
	outputMap["timestamp"] = &timestamp

	// TODO: use a faster marshaller
	data, _ := json.Marshal(outputMap) //nolint:errcheck // errors can't pop up here
	return data
}
