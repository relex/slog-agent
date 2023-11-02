package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanUTF8(t *testing.T) {
	assert.Equal(t, "", string(CleanUTF8([]byte(""))))
	assert.Equal(t, "Test брэд-ЛГТМ", string(CleanUTF8([]byte("Test брэд-ЛГТМ"))))
	assert.Equal(t, "世界", string(CleanUTF8([]byte("世界"[:6]))))
	assert.Equal(t, "世", string(CleanUTF8([]byte("世界"[:5]))))
	assert.Equal(t, "世", string(CleanUTF8([]byte("世界"[:4]))))
	assert.Equal(t, "世", string(CleanUTF8([]byte("世界"[:3]))))
	assert.Equal(t, "", string(CleanUTF8([]byte("世界"[:2]))))
	assert.Equal(t, "Hello, 世界", string(CleanUTF8([]byte("Hello, 世界"[:13]))))
	assert.Equal(t, "Hello, 世", string(CleanUTF8([]byte("Hello, 世界"[:12]))))
	assert.Equal(t, "世界 Hi", string(CleanUTF8([]byte("世界 Hi"))))
}
