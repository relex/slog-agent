package base

// LogCustomCounterRegistry allows registration of custom record counters by label
//
// RegisterCustomCounter returns a function to be called to count record length
type LogCustomCounterRegistry interface {
	RegisterCustomCounter(label string) func(length int)
}
