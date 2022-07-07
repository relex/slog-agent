package fluentdforward

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/output/fastmsgpack"
	"golang.org/x/exp/slices"
)

type eventSerializer struct {
	logger                 logger.Logger
	schema                 base.LogSchema
	fieldMasks             []bool                 // mark environment fields and hidden fields, same length as LogRecords.Fields
	envFieldLocators       []base.LogFieldLocator // locators of environment fields
	fieldRewriters         []base.LogRewriter     // head writers for each of fields or nil, same length as LogRecords.Fields
	serializedFieldKeys    []msgpackBlock         // pre-serialized field keys
	serializedEnvFieldKeys []msgpackBlock         // pre-serialized environment field keys
	buffer                 []byte                 // use fastmsgpack+array is 2.5x faster than msgpack+bytes.Buffer due to massive inlining
	deallocator            *base.LogAllocator
}

type msgpackBlock []byte

// MustNewEventSerializer creates ForwardLogEventSerializer or panic
func MustNewEventSerializer(parentLogger logger.Logger, schema base.LogSchema, config SerializationConfig,
	allocator *base.LogAllocator) base.LogSerializer {

	s, err := NewEventSerializer(parentLogger, schema, config, allocator)
	if err != nil {
		logger.Panic("failed to create FluentdForwardEventSerializer: ", err)
	}
	return s
}

// NewEventSerializer creates a LogSerializer to serialize log records into fluentd's MessagePackEventStream (as ForwardMessage.Entries)
// MessagePackEventStream is a sequence of log records in msgpack format: [timestamp, map of fields]
// "environment" map is nested inside the map of fields
func NewEventSerializer(parentLogger logger.Logger, schema base.LogSchema, config SerializationConfig,
	deallocator *base.LogAllocator) (base.LogSerializer, error) {

	fieldNames := schema.GetFieldNames()

	envFieldLocators := make([]base.LogFieldLocator, len(config.EnvironmentFields))
	for i, name := range config.EnvironmentFields {
		loc, lerr := schema.CreateFieldLocator(name)
		if lerr != nil {
			return nil, fmt.Errorf("environment field '%s': %w", name, lerr)
		}
		envFieldLocators[i] = loc
	}

	fieldRewriters := make([]base.LogRewriter, len(fieldNames))
	for i, name := range fieldNames {
		rewriterConfigs, ok := config.RewriteFields[name]
		if !ok {
			continue
		}
		fieldRewriters[i] = bsupport.NewRewritersFromConfig(rewriterConfigs, schema)
	}

	fieldMasks := make([]bool, len(fieldNames))
	for i, name := range fieldNames {
		if slices.Index(config.EnvironmentFields, name) != -1 {
			fieldMasks[i] = true
		} else if slices.Index(config.HiddenFields, name) != -1 {
			fieldMasks[i] = true
		}
	}

	return &eventSerializer{
		logger:                 parentLogger.WithField(defs.LabelComponent, "FluentdForwardEventSerializer"),
		schema:                 schema,
		fieldMasks:             fieldMasks,
		envFieldLocators:       envFieldLocators,
		fieldRewriters:         fieldRewriters,
		serializedFieldKeys:    preSerializeStrings(fieldNames),
		serializedEnvFieldKeys: preSerializeStrings(config.EnvironmentFields),
		buffer:                 make([]byte, 2*defs.InputLogMaxMessageBytes),
		deallocator:            deallocator,
	}, nil
}

// SerializeRecord serializes log records into streams
func (packer *eventSerializer) SerializeRecord(record *base.LogRecord) base.LogStream {
	len := packer.encodeRecord(record, packer.buffer)
	packer.deallocator.Release(record)
	return packer.buffer[:len]
}

// encodeRecord encodes the given log record to buffer and returns the end position
// DO NOT deduplicate the code below - they're required for go inlining to work! (as of v1.15)
// Check with `disasm` command in go pprof to be sure
func (packer *eventSerializer) encodeRecord(record *base.LogRecord, buffer []byte) int {
	// encode log records into chunks of [timestamp, field-map] in msgpack
	fields := record.Fields[0:len(packer.fieldMasks)] // hide unnamed/reserved fields at the end
	position := 0

	// root-array
	position = fastmsgpack.EncodeArrayLen4(buffer, position, 2)
	// root-array[0]: timestamp
	position = EncodeEventTime(buffer, position, record.Timestamp)
	// root-array[1]: field-map length
	reservedRootMapLenPosition := position // space reserved, to be encoded later
	switch {
	case len(fields)+1 < 16:
		position = fastmsgpack.ReserveLen4(position)
	default:
		position = fastmsgpack.ReserveLen16(position)
	}

	// root-array[1]: field-map key-value pairs
	rootMapSize := 1 // +1 for nested "environment" map
	{
		fieldMasks := packer.fieldMasks
		serializedFieldKeys := packer.serializedFieldKeys
		fieldRewriters := packer.fieldRewriters

		for i, value := range fields {
			if fieldMasks[i] || len(value) == 0 {
				continue
			}

			// encode field key (pre-serialized)
			position += copy(buffer[position:], serializedFieldKeys[i])

			// encode field value
			headRewriter := fieldRewriters[i]
			if headRewriter != nil {
				// encode rewritten field length
				reservedLengthPosition := position
				maxLength := headRewriter.MaxFieldLength(value, record)
				switch {
				case maxLength < 65536:
					position = fastmsgpack.EncodeStringLen16(buffer, position, maxLength)
				default:
					position = fastmsgpack.EncodeStringLen32(buffer, position, maxLength)
				}
				// encode rewritten field contents
				actualLength := headRewriter.WriteFieldBody(value, record, buffer[position:])
				if actualLength != reservedLengthPosition {
					switch {
					case maxLength < 65536: // use same length type as reserved, not actual
						fastmsgpack.EncodeStringLen16(buffer, reservedLengthPosition, actualLength)
					default:
						fastmsgpack.EncodeStringLen32(buffer, reservedLengthPosition, actualLength)
					}
				}
				position += actualLength
			} else {
				switch {
				case len(value) < 16:
					position = fastmsgpack.EncodeString4(buffer, position, value)
				case len(value) < 65536:
					position = fastmsgpack.EncodeString16(buffer, position, value)
				default:
					position = fastmsgpack.EncodeString32(buffer, position, value)
				}
			}
			rootMapSize++
		}
	}
	// update root map size
	switch {
	case len(fields)+1 < 16: // use same length type as reserved
		fastmsgpack.EncodeMapLen4(buffer, reservedRootMapLenPosition, rootMapSize)
	default:
		fastmsgpack.EncodeMapLen16(buffer, reservedRootMapLenPosition, rootMapSize)
	}

	// root-array[1]: field-map["environment"] map
	position = fastmsgpack.EncodeString4(buffer, position, "environment")
	{
		envFieldLocators := packer.envFieldLocators
		serializedEnvFieldKeys := packer.serializedEnvFieldKeys

		// encode length of environment map
		switch {
		case len(envFieldLocators) < 16:
			position = fastmsgpack.EncodeMapLen4(buffer, position, len(envFieldLocators))
		default:
			position = fastmsgpack.EncodeMapLen16(buffer, position, len(envFieldLocators))
		}

		// root-array[1]: field-map["environment"] map key-value pairs
		for i, loc := range envFieldLocators {
			// encode key (pre-serialized)
			position += copy(buffer[position:], serializedEnvFieldKeys[i])
			// encode value
			value := loc.Get(fields)
			switch {
			case len(value) < 16:
				position = fastmsgpack.EncodeString4(buffer, position, value)
			case len(value) < 65536:
				position = fastmsgpack.EncodeString16(buffer, position, value)
			default:
				position = fastmsgpack.EncodeString32(buffer, position, value)
			}
		}
	}

	if position == len(buffer) {
		packer.logger.Errorf("serialized log exceeds buffer limit: %d", position)
		return 0
	}
	return position
}

func preSerializeStrings(strValues []string) []msgpackBlock {
	results := make([]msgpackBlock, len(strValues))

	for index, key := range strValues {
		buf := make(msgpackBlock, 5+len(key))
		end := 0
		switch {
		case len(key) < 16:
			end = fastmsgpack.EncodeString4(buf, 0, key)
		case len(key) < 65536:
			end = fastmsgpack.EncodeString16(buf, 0, key)
		default:
			end = fastmsgpack.EncodeString32(buf, 0, key)
		}
		results[index] = buf[:end]
	}

	return results
}
