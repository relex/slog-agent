package bmatch

import (
	"testing"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

type LogMatcherTestData struct {
	Match LogMatcherConfig `yaml:"match"`
}

func TestLogMatch(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"facility", "level", "log", "task", "source"})
	d := &LogMatcherTestData{}
	if !assert.NoError(t, util.UnmarshalYamlString(`
match:
  facility: !!regex (local[0-9]|syslog)
  level: !!str-not DEBUG
  log: !!str-contain FooBar
  task: !!str-any
  source: !!str-contain .log
`, d)) {
		return
	}
	lm := d.Match.NewMatcher(schema)
	t.Run("test order by cost", func(tt *testing.T) {
		assert.Equal(t, "task", lm.fieldMatches[0].locator.Name(schema))
		assert.Equal(t, "level", lm.fieldMatches[1].locator.Name(schema))
		assert.Equal(t, "source", lm.fieldMatches[2].locator.Name(schema))
		assert.Equal(t, "log", lm.fieldMatches[3].locator.Name(schema))
		assert.Equal(t, "facility", lm.fieldMatches[4].locator.Name(schema))
	})

	assert.True(t, lm.Match(schema.NewTestRecord1(base.LogFields{"syslog", "WARN", "Foo FooBar", "10A", "task.log"})))
	assert.False(t, lm.Match(schema.NewTestRecord1(base.LogFields{"syslog", "DEBUG", "Foo FooBar", "105", "task.log"})))
	assert.False(t, lm.Match(schema.NewTestRecord1(base.LogFields{"syslog", "", "", "", ""})))
}
