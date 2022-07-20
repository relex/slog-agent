package base

import (
	"fmt"
	"strings"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/util"
	"github.com/relex/slog-agent/util/stringtemplate"
	"golang.org/x/exp/slices"
)

// LogSchema defines the field names and mark fields which should belong to "environment"
// In case of runtime schema update, only new fields should be appended at the end
type LogSchema struct {
	fieldNames []string
	maxFields  int

	OnLocated func(index int) // optional callback invoked after sucessful CreateFieldLocator calls
}

// MustNewLogSchema creates a new LogSchema or panic
func MustNewLogSchema(fieldNames []string) LogSchema {
	schema, err := NewLogSchema(fieldNames, len(fieldNames))
	if err != nil {
		logger.Panic("failed to create schema: ", err)
	}
	return schema
}

// NewLogSchema creates a new LogSchema with field names and environment field names
func NewLogSchema(fieldNames []string, maxFields int) (LogSchema, error) {
	if maxFields < len(fieldNames) {
		return LogSchema{}, fmt.Errorf("maxFields (%d) must be equal or greater than the number of field names (%d)", maxFields, len(fieldNames))
	}

	m := make(map[string]bool, len(fieldNames)*2)
	for i, name := range fieldNames {
		if len(name) == 0 {
			return LogSchema{}, fmt.Errorf("invalid %dth field '%s'", i, name)
		}
		_, exists := m[name]
		if exists {
			return LogSchema{}, fmt.Errorf("duplicated %dth field '%s'", i, name)
		}
		m[name] = true
	}
	schema := LogSchema{
		fieldNames: fieldNames,
		maxFields:  maxFields,
		OnLocated:  nil,
	}
	return schema, nil
}

// CopyTestRecord makes a deep copy of given record
func (s *LogSchema) CopyTestRecord(source *LogRecord) *LogRecord {
	dest := newLogRecord(s.maxFields)
	dest._refCount++
	dest.Timestamp = source.Timestamp
	dest.Unescaped = source.Unescaped
	dest.Fields = util.DeepCopyStrings(source.Fields)
	return dest
}

// NewTestRecord1 creates new record with initial field values for  testing
func (s *LogSchema) NewTestRecord1(fields LogFields) *LogRecord {
	if len(fields) != len(s.fieldNames) {
		logger.Panicf("wrong numbers of test log fields: %s, should be %d", fields, len(s.fieldNames))
	}
	record := newLogRecord(s.maxFields)
	record._refCount++
	record.Fields = fields
	return record
}

// NewTestRecord2 creates new record with initial timestamp and field values for  testing
func (s *LogSchema) NewTestRecord2(tm time.Time, fields LogFields) *LogRecord {
	if len(fields) != len(s.fieldNames) {
		logger.Panicf("wrong numbers of test log fields: %s, should be %d", fields, len(s.fieldNames))
	}
	record := newLogRecord(s.maxFields)
	record._refCount++
	record.Timestamp = tm
	record.Fields = fields
	return record
}

// CreateFieldLocator creates a LogFieldLocator by field name
func (s *LogSchema) CreateFieldLocator(name string) (LogFieldLocator, error) {
	index := slices.Index(s.fieldNames, name)
	if index == -1 {
		return MissingFieldLocator, fmt.Errorf("field '%s' is not defined in schema", name)
	}
	if cb := s.OnLocated; cb != nil {
		cb(index)
	}
	return LogFieldLocator(index), nil
}

// CreateFieldLocators creates LogFieldLocator(s) for field names
func (s *LogSchema) CreateFieldLocators(names []string) ([]LogFieldLocator, error) {
	locators := make([]LogFieldLocator, len(names))
	for i, name := range names {
		loc, err := s.CreateFieldLocator(name)
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", i, err)
		}
		locators[i] = loc
	}
	return locators, nil
}

// CreateTemplateVariableResolver creates a variable resolver by field name, to be used with stringtemplate.StringTemplate
func (s *LogSchema) CreateTemplateVariableResolver(name string) (stringtemplate.PartProvider, error) {
	locator, err := s.CreateFieldLocator(name)
	if err != nil {
		return nil, err
	}
	return locator.provideTemplatePart, nil
}

// GetFieldNames returns all the field names in the same order
func (s *LogSchema) GetFieldNames() []string {
	return s.fieldNames
}

// GetMaxFields returns the maximum number of fields
//
// Max is equal or greater than the numbers of actual fields for reservation
func (s *LogSchema) GetMaxFields() int {
	return s.maxFields
}

// MustCreateFieldLocator creates LogFieldLocator by field name or panic (if field doesn't exist in schema)
func (s *LogSchema) MustCreateFieldLocator(name string) LogFieldLocator {
	loc, err := s.CreateFieldLocator(name)
	if err != nil {
		logger.Panicf("failed to create locator for field [%s]: %s", name, err.Error())
	}
	return loc
}

// MustCreateFieldLocators creates LogFieldLocators for field names or panic (if a field doesn't exist in schema)
func (s *LogSchema) MustCreateFieldLocators(names []string) []LogFieldLocator {
	locs, err := s.CreateFieldLocators(names)
	if err != nil {
		logger.Panicf("failed to create locators for fields [%s]: %s", strings.Join(names, ","), err.Error())
	}
	return locs
}
