package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopySlice(t *testing.T) {
	s1 := []string{"a", "b", "c"}
	s2 := CopySlice(s1)

	assert.Equal(t, []string{"a", "b", "c"}, s2)
	s1[1] = "b2"
	assert.Equal(t, []string{"a", "b", "c"}, s2)
}
