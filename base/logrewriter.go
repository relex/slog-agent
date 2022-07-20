package base

// LogRewriter rewrites the value of specific field during log serialization
// It's a special form of LogTransform, per-field and without filtering, meant to reduce heap allocations.
// For example, inlining of component names to the beginning of log messages can be re-implemented there,
// writing directly to the buffer for serialized log records without any string concentration
// Each LogRewriter takes the field value, the record and the next LogRewriter
type LogRewriter interface {

	// MaxFieldLength returns the maximum possible length of the rewritten value
	MaxFieldLength(value string, record *LogRecord) int

	// WriteFieldBody writes field value to the beginning of given buffer and returns the end position
	WriteFieldBody(value string, record *LogRecord, buffer []byte) int
}
