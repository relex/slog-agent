package base

// LogStream represents serialized log record. Each LogStream should contain exactly one record.
// The data should be uncompressed and temporary
type LogStream []byte
