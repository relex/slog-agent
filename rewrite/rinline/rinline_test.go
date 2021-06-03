package rinline

import (
	"testing"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/rewrite/rcopy"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestInlineRewrite(t *testing.T) {
	schema := base.MustNewLogSchema([]string{"id", "msg"})
	{
		c := &Config{}
		assert.Nil(t, util.UnmarshalYamlString(`
type: inline
field: id
`, c))
		rw := c.NewRewriter(schema, rcopy.NewRewriter())
		msg := "message"
		result := "id=foo message"
		record := schema.NewTestRecord1(base.LogFields{"foo", msg})
		buf := make([]byte, 100)
		assert.Equal(t, len(result), rw.MaxFieldLength(msg, record))
		assert.Equal(t, len(result), rw.WriteFieldBody(msg, record, buf))
		assert.Equal(t, result, string(buf[:len(result)]))
	}
}
