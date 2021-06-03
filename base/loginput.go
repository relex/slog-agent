package base

// LogInput represents an input source in the beginning of pipeline, e.g. a TCP/Syslog input
// It integrates endpoint/listener, parser and necessary steps to construct a raw base.LogRecord
type LogInput interface {
	PipelineWorker
	Address() string
}
