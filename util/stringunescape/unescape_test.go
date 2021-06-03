package stringunescape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnescape(t *testing.T) {
	u := NewUnescaper('\\', map[byte]byte{
		'*': '*',
		'O': 'O',
		'b': '\b',
		'f': '\f',
		'n': '\n',
		'r': '\r',
		't': '\t',
	})
	assert.Equal(t, "hello\nworld\n123", u.Run(`hello\nworld\n123`))
	assert.Equal(t, "\nhello\nworld", u.Run(`\nhello\nworld`))
	assert.Equal(t, "x\\Xhello\n", u.Run(`x\Xhello\n`), "Unescape invalid escape character")
	assert.Equal(t, "x\n\n\txx\n", u.Run(`x\n\n\txx\n`))
	assert.Equal(t, "x\n\n\txx\\N", u.Run(`x\n\n\txx\N`), "Unescape invalid escape char at end")
	assert.Equal(t, "x\n\n\txx\\", u.Run(`x\n\n\txx\`), "Unescape trailing backslash")
	assert.Equal(t, "x\n\n\txx\\", u.Run(`x\n\n\txx\\`))
	assert.Equal(t, "x\n\n\txx\\\n", u.Run(`x\n\n\txx\\\n`))
	t.Run("find unescaped",
		func(tt *testing.T) {
			assert.Equal(t, 2, u.FindFirstUnescaped("he**o", '*'))
			assert.Equal(t, -1, u.FindFirstUnescaped("hello", 'O'))
			assert.Equal(t, 6, u.FindFirstUnescaped(`h\*ell*`, '*'))
			assert.Equal(t, 2, u.FindFirstUnescaped(`\**`, '*'))
		})
}
