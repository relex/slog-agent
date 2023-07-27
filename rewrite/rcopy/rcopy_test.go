package rcopy

import (
	"testing"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestCopyRewrite(t *testing.T) {
	{
		c := &Config{}
		assert.NoError(t, util.UnmarshalYamlString(`
type: copy
`, c))
		rw := c.NewRewriter(base.LogSchema{}, nil)
		msg := "hello"
		buf := make([]byte, 100)
		assert.Equal(t, len(msg), rw.MaxFieldLength(msg, nil))
		assert.Equal(t, len(msg), rw.WriteFieldBody(msg, nil, buf))
		assert.Equal(t, msg, string(buf[:len(msg)]))
	}
}
