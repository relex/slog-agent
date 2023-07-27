package runescape

import (
	"testing"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestUnescapeRewrite(t *testing.T) {
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString(`
type: unescape
`, c))
		rw := c.NewRewriter(base.LogSchema{}, nil)
		rec := &base.LogRecord{}
		msg := `dum\ndum\n.`
		result := "dum\ndum\n."
		buf := make([]byte, 100)
		assert.Equal(t, len(msg), rw.MaxFieldLength(msg, rec))
		assert.Equal(t, len(result), rw.WriteFieldBody(msg, rec, buf))
		assert.Equal(t, result, string(buf[:len(result)]))
	}
}
