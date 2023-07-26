package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogSchema(t *testing.T) {
	_, err1 := NewLogSchema([]string{"a", "b", "c"}, 2)
	assert.ErrorContains(t, err1, "maxFields")

	_, err2 := NewLogSchema([]string{"a", "", "c"}, 10)
	assert.ErrorContains(t, err2, "invalid 1th field")

	_, err3 := NewLogSchema([]string{"a", "b", "b"}, 10)
	assert.ErrorContains(t, err3, "duplicated 2th field 'b'")
}

func TestLogSchema(t *testing.T) {
	schema := MustNewLogSchema([]string{"a", "b"})

	_, err1 := schema.CreateFieldLocator("c")
	assert.ErrorContains(t, err1, "field 'c' is not defined in schema")

	b := schema.MustCreateFieldLocator("b")
	assert.Equal(t, "second", b.Get([]string{"first", "second"}))

	all, errAll := schema.CreateFieldLocators([]string{"b", "a"})
	assert.Nil(t, errAll)
	assert.Equal(t, "second", all[0].Get([]string{"first", "second"}))
	assert.Equal(t, "first", all[1].Get([]string{"first", "second"}))
}
