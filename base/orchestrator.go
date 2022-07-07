package base

// Orchestrator takes log records and distribute them to internal pipelines
type Orchestrator interface {
	MultiSinkBufferReceiver

	// Shutdown performs cleanup; should be called after all inputs have been stopped
	Shutdown()
}
