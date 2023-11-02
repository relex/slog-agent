package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrings(t *testing.T) {
	orig := []byte("hello")
	new1 := StringFromBytes(orig)
	new2 := DeepCopyStringFromBytes(orig)
	orig[0] = 'H'

	assert.Equal(t, "Hello", new1)
	assert.Equal(t, "hello", new2)

	copy := DeepCopyString(new1)
	orig[1] = 'E'
	assert.Equal(t, "HEllo", new1)
	assert.Equal(t, "Hello", copy)

	ary1 := []string{new1, "world"}
	ary2 := DeepCopyStrings(ary1)
	orig[4] = 'O'
	assert.Equal(t, []string{"HEllO", "world"}, ary1)
	assert.Equal(t, []string{"HEllo", "world"}, ary2)

	new3 := BytesFromString(new1)
	new3[2] = 'X'
	assert.Equal(t, "HEXlO", string(new3))
	assert.Equal(t, "HEXlO", new1)
}

func TestStringOverwrite(t *testing.T) {
	assert.Equal(t, "helloABC", string(OverwriteNTruncate([]byte("helloWorld"), 5, "ABC")))
	assert.Equal(t, "hell", string(OverwriteNTruncate([]byte("helloABC"), 4, "")))
	assert.Equal(t, "hel^-", string(OverwriteNTruncate([]byte("hell."), 3, "^-^")))
}
