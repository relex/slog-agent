package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanUTF8(t *testing.T) {
	assert.Equal(t, "", CleanUTF8(""))
	assert.Equal(t, "Test брэд-", CleanUTF8("Test брэд-ЛГТМ"))
	assert.Equal(t, "", CleanUTF8("世界"))
	assert.Equal(t, "Hello, ", CleanUTF8("Hello, 世界"))
	assert.Equal(t, "世界 Hi", CleanUTF8("世界 Hi"))
}
