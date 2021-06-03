package bmatch

import (
	"fmt"
	"testing"

	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

type valueMatchTestData struct {
	Value valueMatch `yaml:"value"`
}

func TestValueMatch(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!str my-name`); f != nil {
		assert.True(t, f("my-name"))
		assert.False(t, f("hello"))
	}

	if f := tryBuildMatch(t, `value: !!str-eq yes`); f != nil {
		assert.True(t, f("yes"))
		assert.False(t, f("no"))
	}
}

func TestValueMatchUnsupportedTag(t *testing.T) {
	d := &valueMatchTestData{}
	assert.Equal(t, fmt.Errorf("yaml line 1:8: Unsupported value-match tag: !!hello"), util.UnmarshalYamlString(`value: !!hello my-name`, d))
}

func TestValueMatchNot(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!str-not yes`); f != nil {
		assert.True(t, f("no"))
		assert.False(t, f("yes"))
	}
}

func TestValueMatchStart(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!str-start Foo`); f != nil {
		assert.True(t, f("FooBar"))
		assert.False(t, f("[Foo]"))
	}
}

func TestValueMatchEnd(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!str-end Bar`); f != nil {
		assert.True(t, f("FooBar"))
		assert.False(t, f("[Bar]"))
	}
}

func TestValueMatchContain(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!str-contain Ok`); f != nil {
		assert.True(t, f("[Ok]"))
		assert.False(t, f("O.k"))
	}
}

func TestValueMatchGlob(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!glob P[OU][ST]** params=**`); f != nil {
		assert.False(t, f(`GET "/logs", status=200 params={"format":"json""since":"1606910669640.002"}`))
		assert.True(t, f(`PUT "/new", status=201 params={"name": "new entry"}`))
	}
}

func TestValueMatchRegex(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!regex ^Hello.*World$`); f != nil {
		assert.True(t, f("HelloXXXWorld"))
		assert.False(t, f("HelloXXX"))
	}
}

func TestValueMatchRegexInvalid(t *testing.T) {
	d := &valueMatchTestData{}
	e := util.UnmarshalYamlString(`value: !!regex ^Hello[.*World`, d)
	if assert.NotNil(t, e) {
		assert.Contains(t, e.Error(), "yaml line 1:8: Failed value-match of tag !!regex: error parsing regexp: ")
	}
}

func TestValueMatchLengthGreaterThan(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!len-gt 10`); f != nil {
		assert.True(t, f("1234567890A"))
		assert.False(t, f("1234567890"))
	}
}

func TestValueMatchLengthLessThan(t *testing.T) {
	if f := tryBuildMatch(t, `value: !!len-lt 5`); f != nil {
		assert.True(t, f("abcd"))
		assert.False(t, f("abcde"))
	}
}

func tryBuildMatch(t *testing.T, matcherYAML string) valueMatcher {
	d := &valueMatchTestData{}

	if assert.Nil(t, util.UnmarshalYamlString(matcherYAML, d)) {
		return d.Value.match
	}
	return nil
}
