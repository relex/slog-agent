package base

// LogRecordBatch represents a batch of logs passed from orchestrators to transformation pipelines
type LogRecordBatch struct {
	Records []*LogRecord
	Full    bool // true if the batch is sent due to buffer limit reached (as opposed to periodic flushing)
}
