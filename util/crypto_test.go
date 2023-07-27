package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMD5ToHexdigest(t *testing.T) {
	assert.Equal(t, "8b1a9953c4611296a827abf8c47804d7", MD5ToHexdigest("Hello"))
}

func TestSHA512ToHexdigest(t *testing.T) {
	assert.Equal(t, "3615f80c9d293ed7402687f94b22d58e529b8cc7916f8fac7fddf7fbd5af4cf777d3d795a7a00a16bf7e7f3fb9561ee9baae480da9fe7a18769e71886b03f315", SHA512ToHexdigest("Hello"))
}
