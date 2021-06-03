package base

// LogListener represents an input endpoint for logs, e.g. a file listener or TCP listener
// Due to data complexity, the output is to be received by a MultiChannelReceiver passed during construction
// LogListener always works in background as one or more goroutines
type LogListener interface {
	PipelineWorker
}
