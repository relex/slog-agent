package bsupport

import (
	"github.com/relex/slog-agent/base"
)

type stubLogCustomCounterRegistry struct {
}

// NewStubLogCustomCounterRegistry creates a stub LogCustomCounterRegistry for testing
func NewStubLogCustomCounterRegistry() base.LogCustomCounterRegistry {
	return &stubLogCustomCounterRegistry{}
}

func (stub *stubLogCustomCounterRegistry) RegisterCustomCounter(label string) func(length int) {
	return func(length int) {}
}
