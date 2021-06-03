package sysloginput

import (
	"time"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
)

type compositeParser struct {
	underlyingParser     base.LogParser
	extractionTransforms []base.LogTransformFunc
	deallocator          *base.LogAllocator
}

// newCompositeParser combines a parser and a set of extraction transforms that are executed immediately after parsing without additional goroutine
func newCompositeParser(p base.LogParser, extractions []base.LogTransformFunc, deallocator *base.LogAllocator) base.LogParser {
	return &compositeParser{
		underlyingParser:     p,
		extractionTransforms: extractions,
		deallocator:          deallocator,
	}
}

func (cp *compositeParser) Parse(input []byte, timestamp time.Time) *base.LogRecord {
	record := cp.underlyingParser.Parse(input, timestamp)
	if record == nil {
		// parser should log details by itself
		return nil
	}
	if bsupport.RunTransforms(record, cp.extractionTransforms) == base.DROP {
		cp.deallocator.Release(record)
		// TODO: metrics
		return nil
	}
	return record
}
