package btest

import (
	"fmt"

	"github.com/relex/slog-agent/base"
)

type stubLogCustomCounterRegistry struct {
	lookup map[string]*stubLogCustomCounter
}

type stubLogCustomCounter struct {
	count  int64
	length int64
}

// LookupStubCustomerCounterFunc can be called to get (log count, total log length) by registered label name
//
// For testing only
type LookupStubCustomerCounterFunc func(label string) (int64, int64)

// NewStubLogCustomCounterRegistry creates a stub LogCustomCounterRegistry for testing
func NewStubLogCustomCounterRegistry() (base.LogCustomCounterRegistry, LookupStubCustomerCounterFunc) {
	reg := &stubLogCustomCounterRegistry{
		lookup: make(map[string]*stubLogCustomCounter),
	}
	return reg, reg.lookupCustomCounter
}

func (stub *stubLogCustomCounterRegistry) RegisterCustomCounter(label string) func(length int) {
	var pCounter *stubLogCustomCounter
	var exists bool

	pCounter, exists = stub.lookup[label]
	if !exists {
		pCounter = &stubLogCustomCounter{}
		stub.lookup[label] = pCounter
	}

	return func(length int) {
		pCounter.count++
		pCounter.length += int64(length)
	}
}

func (stub *stubLogCustomCounterRegistry) lookupCustomCounter(label string) (int64, int64) {
	pCounter, exists := stub.lookup[label]
	if !exists {
		panic(fmt.Sprintf("counter %s does not exist", label))
	}

	return pCounter.count, pCounter.length
}
