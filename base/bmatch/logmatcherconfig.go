package bmatch

import (
	"fmt"
	"sort"

	"github.com/relex/slog-agent/base"
)

// LogMatcherConfig is the configuration for LogMatch, to match log records of certain conditions
type LogMatcherConfig map[string]valueMatch

type sortableKeyValueMatches struct {
	data []unsortedKeyValueMatch
}

type unsortedKeyValueMatch struct {
	locator base.LogFieldLocator
	match   valueMatch
}

// NewMatcher creates a LogMatcher from a set of field names and ValueMatch(s)
func (cmap LogMatcherConfig) NewMatcher(schema base.LogSchema) LogMatcher {
	ufieldMatches := make([]unsortedKeyValueMatch, 0, len(cmap))
	for key, match := range cmap {
		loc := schema.MustCreateFieldLocator(key)
		ufieldMatches = append(ufieldMatches, unsortedKeyValueMatch{locator: loc, match: match})
	}
	sort.Sort(sortableKeyValueMatches{ufieldMatches})

	sfieldMatches := make([]keyValueMatch, 0, len(cmap))
	for _, pair := range ufieldMatches {
		sfieldMatches = append(sfieldMatches, keyValueMatch{locator: pair.locator, match: pair.match.match})
	}
	return LogMatcher{sfieldMatches}
}

// VerifyConfig checks all field names
func (cmap LogMatcherConfig) VerifyConfig(schema base.LogSchema) error {
	for key, matcher := range cmap {
		_, err := schema.CreateFieldLocator(key)
		if err != nil {
			return fmt.Errorf("invalid match key '%s': %w", key, err)
		}
		if matcher.match == nil { // extra check because empty value in map would NOT go through unmarshalling
			return fmt.Errorf("missing match value for '%s'", key)
		}
	}
	return nil
}

func (s sortableKeyValueMatches) Len() int {
	return len(s.data)
}

func (s sortableKeyValueMatches) Less(i, j int) bool {
	return s.data[i].match.cost < s.data[j].match.cost
}

func (s sortableKeyValueMatches) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}
