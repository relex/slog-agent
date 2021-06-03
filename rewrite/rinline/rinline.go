// Package rinline provides 'inline' rewriter, which inlines exactly one other field into the current field value if exists
//
// It's a specialized version of 'addField' transform, e.g. "message: class=$class $message"
package rinline

import (
	"fmt"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for inlineRewriter
type Config struct {
	bconfig.Header `yaml:",inline"`
	Field          string `yaml:"field"`
}

type inlineRewriter struct {
	header       string
	separator    string
	fieldLocator base.LogFieldLocator
	next         base.LogRewriter
}

// NewRewriter creates inlineRewriter
func (c *Config) NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter {
	if next == nil {
		panic("'inline' cannot be the last rewriter")
	}
	locator := schema.MustCreateFieldLocator(c.Field)
	return &inlineRewriter{
		header:       locator.Name(schema) + "=",
		separator:    " ",
		fieldLocator: locator,
		next:         next,
	}
}

// VerifyConfig verifies inlineRewriter config
func (c *Config) VerifyConfig(schema base.LogSchema, hasNext bool) error {
	if !hasNext {
		return fmt.Errorf("'inline' cannot be the last rewriter")
	}
	if len(c.Field) == 0 {
		return fmt.Errorf(".field is unspecified")
	}
	if _, err := schema.CreateFieldLocator(c.Field); err != nil {
		return fmt.Errorf(".field '%s' is invalid: %w", c.Field, err)
	}
	return nil
}

func (rw *inlineRewriter) MaxFieldLength(value string, record *base.LogRecord) int {
	fieldValue := rw.fieldLocator.Get(record.Fields)
	if len(fieldValue) > 0 {
		return len(rw.header) + len(fieldValue) + len(rw.separator) + rw.next.MaxFieldLength(value, record)
	}
	return rw.next.MaxFieldLength(value, record)
}

func (rw *inlineRewriter) WriteFieldBody(value string, record *base.LogRecord, buffer []byte) int {
	fieldValue := rw.fieldLocator.Get(record.Fields)
	if len(fieldValue) > 0 {
		end := copy(buffer, rw.header)
		end += copy(buffer[end:], rw.fieldLocator.Get(record.Fields))
		end += copy(buffer[end:], rw.separator)
		return end + rw.next.WriteFieldBody(value, record, buffer[end:])
	}
	return rw.next.WriteFieldBody(value, record, buffer)
}
