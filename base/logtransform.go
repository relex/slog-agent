package base

// LogTransform filters and/or transforms logs one by one
// May have persistent states
type LogTransform interface {

	// Transform transforms the given record and returns PASS or DROP in addition to the new log record
	// The returned record may share children with the input, modified in-place.
	// The input record shall not be used afterwards
	Transform(input *LogRecord) FilterResult
}

// LogTransformFunc defines a function to perform transformation on a single log record
type LogTransformFunc func(input *LogRecord) FilterResult

// FilterResult defines the result of filtering, pass (true) or drop (false)
type FilterResult bool

// PASS means transform succeeds (whether there is change or not)
const PASS FilterResult = true

// DROP means transform aborts and the record is to be dropped
const DROP FilterResult = false
