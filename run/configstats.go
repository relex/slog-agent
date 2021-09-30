package run

import (
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
)

// ConfigStats provides extra stats related to quality of config file
type ConfigStats struct {
	FixedFields       []string // Fixed fields are those used by inputs and orchestration that cannot be moved or renamed
	UnusedFields      []string
	OrchestrationKeys []string
}

// Log logs important information or warnings if there is any
func (stats ConfigStats) Log(logger logger.Logger) {
	if len(stats.UnusedFields) > 0 {
		logger.Warn("unused fields in config: ", stats.UnusedFields)
	}
}

// ConfigStatsBuilder helps building ConfigStats
type ConfigStatsBuilder struct {
	schema      *base.LogSchema
	fieldsFixed []bool
	fieldsInUse []bool
}

// NewConfigStatsBuilder creates a ConfigStatsTracker for the given schema
//
// Hook(s) in the schema will be used for tracking, and as a result a schema must have multiple trackers associated
func NewConfigStatsBuilder(schema *base.LogSchema) *ConfigStatsBuilder {
	return &ConfigStatsBuilder{
		schema:      schema,
		fieldsFixed: make([]bool, len(schema.GetFieldNames())),
		fieldsInUse: make([]bool, len(schema.GetFieldNames())),
	}
}

// BeginTrackingFixedFields begins tracking of fields that must remain fixed during config reload, not removed or moved
func (tracker *ConfigStatsBuilder) BeginTrackingFixedFields() {
	tracker.schema.OnLocated = func(index int) {
		tracker.fieldsFixed[index] = true
		tracker.fieldsInUse[index] = true
	}
}

// BeginTrackingFields begins tracking of all fields for their usage
func (tracker *ConfigStatsBuilder) BeginTrackingFields() {
	tracker.schema.OnLocated = func(index int) {
		tracker.fieldsInUse[index] = true
	}
}

// Finish ends all tracking and exports results to the given ConfigStats
func (tracker *ConfigStatsBuilder) Finish(confStats *ConfigStats) {
	tracker.schema.OnLocated = nil

	fixedFields := make([]string, 0, len(tracker.fieldsFixed))
	unusedFields := make([]string, 0, len(tracker.fieldsInUse))

	fieldNames := tracker.schema.GetFieldNames()
	for i, fixed := range tracker.fieldsFixed {
		if fixed {
			fixedFields = append(fixedFields, fieldNames[i])
		}
	}
	for i, inUse := range tracker.fieldsInUse {
		if !inUse {
			unusedFields = append(unusedFields, fieldNames[i])
		}
	}

	confStats.FixedFields = fixedFields
	confStats.UnusedFields = unusedFields
}
