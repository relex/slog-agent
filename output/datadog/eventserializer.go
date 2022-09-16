package datadog

import (
	"encoding/json"
	"strconv"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

type eventSerializer struct {
	logger     logger.Logger
	config     SerializationConfig
	schema     base.LogSchema
	fieldMasks []bool // mark hidden fields, same length as LogRecords.Fields
	ddtags     string
}

func NewEventSerializer(parentLogger logger.Logger, schema base.LogSchema, config SerializationConfig, ddtags string) base.LogSerializer {
	return &eventSerializer{
		logger: parentLogger,
		schema: schema,
		config: config,
		fieldMasks: lo.Map(schema.GetFieldNames(), func(fieldName string, _ int) bool {
			return len(fieldName) == 0 || slices.Contains(config.HiddenFields, fieldName)
		}),
		ddtags: ddtags,
	}
}

func (packer *eventSerializer) SerializeRecord(record *base.LogRecord) base.LogStream {
	fieldNames := packer.schema.GetFieldNames()
	outputMap := make(map[string]string, len(fieldNames)+4)

	for i, fieldName := range fieldNames {
		if !packer.fieldMasks[i] && record.Fields[i] != "" {
			outputMap[fieldName] = record.Fields[i]
		}
	}

	outputMap["timestamp"] = strconv.FormatInt(record.Timestamp.UnixMilli(), 10)

	if len(outputMap["ddtags"]) == 0 && len(packer.ddtags) > 0 {
		outputMap["ddtags"] = packer.ddtags
	}

	// TODO: might consider using a faster marshaller such as go-json
	data, _ := json.Marshal(outputMap) //nolint:errcheck // errors can't pop up here
	return data
}
