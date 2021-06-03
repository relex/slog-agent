package bmatch

import (
	"github.com/relex/slog-agent/base"
)

// LogMatcher can match log records of certain conditions
type LogMatcher struct {
	fieldMatches []keyValueMatch
}

type keyValueMatch struct {
	locator base.LogFieldLocator
	match   valueMatcher
}

// Match checks whether the given record matches the condition of this matcher
func (m LogMatcher) Match(record *base.LogRecord) bool {
	fields := record.Fields
	// TODO: REUSE results from distribution if all keys are among the distribution's key fields
	for _, fm := range m.fieldMatches {
		value := fm.locator.Get(fields)
		if !fm.match(value) {
			return false
		}
	}
	return true
}
