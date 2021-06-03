package stringtemplate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringTemplate(t *testing.T) {
	resolveVariable := func(name string) (PartProvider, error) {
		var index int
		switch name {
		case "pri":
			index = 0
		case "appname":
			index = 1
		case "msgid":
			index = 2
		default:
			return nil, fmt.Errorf("no such field '%s'", name)
		}
		return func(record []string) string {
			return record[index]
		}, nil
	}
	if tmpl, err := NewExpander("mytag-$appname:${msgid}-route0", resolveVariable); assert.Nil(t, err) {
		result := tmpl.Run([]string{"163", "TestParser", "10"})
		assert.Equal(t, "mytag-TestParser:10-route0", result)
	}
	if tmpl, err := NewExpander("mytag-${appname[1:-6]}-", resolveVariable); assert.Nil(t, err) {
		result := tmpl.Run([]string{"4", "TestParser", ""})
		assert.Equal(t, "mytag-est-", result)
	}
	if tmpl, err := NewExpander("mytag-${appname[:3]}-", resolveVariable); assert.Nil(t, err) {
		result := tmpl.Run([]string{"", "ID", ""})
		assert.Equal(t, "mytag-ID-", result)
	}
}

func TestStringTemplate1(t *testing.T) {
	if tmpl, err := NewExpander("nothing", nil); assert.Nil(t, err) {
		assert.Equal(t, 1, len(tmpl.partProviders))
		result := tmpl.Run([]string(nil))
		assert.Equal(t, "nothing", result)
	}
	resolveVariable := func(name string) (PartProvider, error) {
		var index int
		switch name {
		case "num":
			index = 0
		case "key1":
			index = 1
		default:
			return nil, fmt.Errorf("no such field '%s'", name)
		}
		return func(record []string) string {
			return record[index]
		}, nil
	}
	if tmpl, err := NewExpander("${key1[-2:]}", resolveVariable); assert.Nil(t, err) {
		assert.Equal(t, 1, len(tmpl.partProviders))
		result := tmpl.Run([]string{"10", "foo"})
		assert.Equal(t, "oo", result)
	}
}

func TestStringTemplateError(t *testing.T) {
	resolveVariable := func(name string) (PartProvider, error) {
		return nil, nil
	}
	tmpl, err := NewExpander("hello-${field", resolveVariable)
	if !assert.EqualError(t, err, "unenclosed variable quotes: 'hello-${field'") {
		t.Error(tmpl)
	}
}
