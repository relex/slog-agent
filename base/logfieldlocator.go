package base

import (
	"github.com/relex/slog-agent/util"
)

// LogFieldLocator is used to locate a named field in LogRecord, bound to a LogSchema
type LogFieldLocator int

// MissingFieldLocator represents non-existing index to a log field
const MissingFieldLocator LogFieldLocator = -1

// Name returns the field name
func (loc LogFieldLocator) Name(schema LogSchema) string { // xx:inline
	return schema.fieldNames[loc]
}

// Get returns the field value or empty string
func (loc LogFieldLocator) Get(fields []string) util.MutableString { // xx:inline
	return fields[loc]
}

// Set assigns the field value
func (loc LogFieldLocator) Set(fields []string, value util.MutableString) { // xx:inline
	fields[loc] = value
}

// Del resets the field value to empty string
func (loc LogFieldLocator) Del(fields []string) { // xx:inline
	fields[loc] = ""
}

// provideTemplatePart is used by LogSchema.CreateTemplateVariableResolver for stringtemplate.StringTemplate
func (loc LogFieldLocator) provideTemplatePart(fields []string) string { // xx:inline
	return fields[loc]
}
