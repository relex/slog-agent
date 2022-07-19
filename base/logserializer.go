package base

// LogSerializer serializes log records into a stream of certain format, e.g. JSON or msgpack
type LogSerializer interface {
	// SerializeRecord serializes the given log record into a log stream
	// Input records should be released in the call
	// Output LogStream is transient and only usable before the next call
	SerializeRecord(record *LogRecord) LogStream
}
