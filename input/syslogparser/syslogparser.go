// Package syslogparser provides a LogParser for Syslog protocol (RFC 5424).
// Timestamps and the contents of "extradata" (metadata) are not parsed. No whitespace is allowed inside "extradata".
//
// Resulting records contain: facility, level, time, host, app, pid, source, extradata (metadata) and log (message)
package syslogparser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/input/syslogprotocol"
	"github.com/relex/slog-agent/util"
)

const maxLoggingMessageSize = 200 // 200 bytes should be enough to include all key fields and the start of message

// syslogparser parses RFC 5424 text to log records
//
// NOT thread-safe due to caching
//
// TODO: parse or skip extradata/metadata properly
type syslogParser struct {
	logger               logger.Logger
	allocator            *base.LogAllocator
	schema               base.LogSchema
	levelMapping         []string
	inputCounter         *base.LogInputCounterSet
	timezoneCache        map[string]*time.Location
	fieldFacilityLocator base.LogFieldLocator
	fieldLevelLocator    base.LogFieldLocator
	fieldLogLocator      base.LogFieldLocator
	restFieldLocators    []base.LogFieldLocator
	overflowCounter      func(length int)
}

// MustNewParser creates a new syslogParser or panic
func MustNewParser(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
	levelMapping []string, inputCounter *base.LogInputCounterSet,
) base.LogParser {
	parser, err := NewParser(parentLogger, allocator, schema, levelMapping, inputCounter)
	if err != nil {
		parentLogger.Panic("failed to create SyslogParser: ", err)
	}

	return parser
}

// NewParser creates a new syslogParser
func NewParser(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
	levelMapping []string, inputCounter *base.LogInputCounterSet,
) (base.LogParser, error) {
	if len(levelMapping) == 0 {
		levelMapping = syslogprotocol.SeverityNames
	} else if len(levelMapping) != 8 {
		return nil, fmt.Errorf("level mapping should have 8 elements not %d", len(levelMapping))
	}

	locFacility, err := schema.CreateFieldLocator("facility")
	if err != nil {
		return nil, err
	}

	locLevel, err := schema.CreateFieldLocator("level")
	if err != nil {
		return nil, err
	}

	locRest := make([]base.LogFieldLocator, 0, 10)
	for _, name := range []string{"time", "host", "app", "pid", "source", "extradata"} {
		loc, serr := schema.CreateFieldLocator(name)
		if serr != nil {
			return nil, serr
		}
		locRest = append(locRest, loc)
	}

	locLog, err := schema.CreateFieldLocator("log")
	if err != nil {
		return nil, err
	}

	parser := &syslogParser{
		logger:               parentLogger.WithField(defs.LabelComponent, "SyslogParser"),
		allocator:            allocator,
		schema:               schema,
		levelMapping:         levelMapping,
		inputCounter:         inputCounter,
		timezoneCache:        make(map[string]*time.Location, 10),
		fieldFacilityLocator: locFacility,
		fieldLevelLocator:    locLevel,
		fieldLogLocator:      locLog,
		restFieldLocators:    locRest,
		overflowCounter:      inputCounter.RegisterCustomCounter("overflow"),
	}

	return parser, nil
}

// Parse parses incoming log and returns (pass, record)
func (parser *syslogParser) Parse(input []byte, timestamp time.Time) *base.LogRecord {
	record, remaining := parser.allocator.NewRecord(input)
	record.RawLength = len(input)
	record.Timestamp = timestamp // actual timestamp is to be parsed and filled by transform.parseTimeTransform

	fields := record.Fields
	if len(remaining) < 32 || remaining[0] != '<' {
		parser.onMalformed(record, "invalid syslog", input)
		return nil
	}

	// parse the pri field, e.g. "<163>"
	ok, val, next := nextFieldBySpace(remaining)
	if !ok {
		parser.onMalformed(record, "unfinished syslog", input)
		return nil
	}

	if val[len(val)-2:] != ">1" {
		parser.onMalformed(record, fmt.Sprintf("invalid syslog pri '%s'", val), input)
		return nil
	}
	pri := val[1 : len(val)-2]
	priVal, err := strconv.Atoi(pri)
	if err != nil {
		parser.onMalformed(record, fmt.Sprintf("invalid syslog pri value '%s'", pri), input)
		return nil
	}

	// extract facility from pri
	facility := priVal >> 3
	if facility < 0 || facility >= len(syslogprotocol.FacilityNames) {
		parser.onMalformed(record, fmt.Sprintf("invalid syslog facility %d", facility), input)
		return nil
	}
	facilityName := syslogprotocol.FacilityNames[facility]
	parser.fieldFacilityLocator.Set(fields, facilityName)

	// extract severity (log level) from pri
	severity := priVal & 0b111
	severityName := parser.levelMapping[severity]
	parser.fieldLevelLocator.Set(fields, severityName)

	remaining = next

	// rest of header fields delimited by whitespace
	for _, locator := range parser.restFieldLocators {
		ok, val, next := nextFieldBySpace(remaining)
		if !ok {
			parser.onMalformed(record, fmt.Sprintf("missing syslog field '%s'", locator.Name(parser.schema)), input)
			return nil
		}
		locator.Set(fields, val)
		remaining = next
	}

	// all the rest of message goes to the "log" message field
	if len(remaining) > defs.InputLogMaxMessageBytes {
		parser.onOverflow(input)
		remaining = remaining[:defs.InputLogMaxMessageBytes]
	}
	parser.fieldLogLocator.Set(fields, remaining)

	// assume the message won't need un-escaping if there is a real newline
	record.Unescaped = strings.IndexByte(remaining, '\n') != -1
	parser.inputCounter.CountRecordPass(record)

	return record
}

func (parser *syslogParser) onMalformed(record *base.LogRecord, warning string, rawLog []byte) {
	parser.inputCounter.CountRecordDrop(record)
	parser.allocator.Release(record)
	// TODO: omit repeated warnings
	if len(rawLog) > maxLoggingMessageSize {
		parser.logger.Warn(warning, ": ", util.StringFromBytes(rawLog[:maxLoggingMessageSize]), "...")
	} else {
		parser.logger.Warn(warning, ": ", util.StringFromBytes(rawLog))
	}
}

func (parser *syslogParser) onOverflow(rawLog []byte) {
	parser.overflowCounter(len(rawLog))
	// TODO: omit repeated warnings
	if len(rawLog) > maxLoggingMessageSize {
		parser.logger.Warn("message overflow: ", util.StringFromBytes(rawLog[:maxLoggingMessageSize]), "...")
	} else {
		parser.logger.Warn("message overflow: ", util.StringFromBytes(rawLog))
	}
}

// nextFieldBySpace takes next field value separated by space
// return (ok, value, remaining part not including space)
// Ex: "a b c" will return (true, "a", "b c")
func nextFieldBySpace(s string) (bool, string, string) {
	end := strings.IndexByte(s, ' ')
	if end == -1 {
		return false, "", ""
	}
	return true, s[:end], s[end+1:]
}
