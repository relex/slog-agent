package tredactemail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactEmail(t *testing.T) {
	t.Run("common", func(tt *testing.T) {
		assert.Equal(t, "trx_key: user_123, user: REDACTED", redactEmail("trx_key: user_123, user: foo.bar@domain.fi"))
		assert.Equal(t, "REDACTED,Yes", redactEmail("foo.bar@domain.fi,Yes"))
		assert.Equal(t, "reply_to: REDACTED,REDACTED,Hello", redactEmail("reply_to: foo-1@domain.fi,foo-2@domain.fi,Hello"))
	})
	t.Run("edge", func(tt *testing.T) {
		assert.Equal(t, "[REDACTEDREDACTEDREDACTED]", redactEmail("[foo-1@domain.fifoo-2@domain.fifoo-3@domain.fi]"))
		assert.Equal(t, "not-email@foo REDACTED something@", redactEmail("not-email@foo a@b.c something@"))
		assert.Equal(t, "@", redactEmail("@"))
		assert.Equal(t, "xxx@", redactEmail("xxx@"))
	})
	t.Run("truncated", func(t *testing.T) {
		assert.Equal(t, "@xxx REDACTED", redactEmail("@xxx something@googl"))
		assert.Equal(t, "truncated REDACTED", redactEmail("truncated something@google."))
	})
	t.Run("not email", func(t *testing.T) {
		assert.Equal(t, "number: hello@123.456", redactEmail("number: hello@123.456"))
		assert.Equal(t, "in Trx@c78b1de/1593788313696 [OPEN]", redactEmail("in Trx@c78b1de/1593788313696 [OPEN]"))
		assert.Equal(t, "in Trx@c78b1de./1593788313696 [OPEN]", redactEmail("in Trx@c78b1de./1593788313696 [OPEN]"))
		assert.Equal(t, "url: ftp://foo:REDACTED", redactEmail("url: ftp://foo:pass@bar.org")) // can't distinguish and password shouldn't show up in logs
		assert.Equal(t, "url: ftp://foo@bar.org", redactEmail("url: ftp://foo@bar.org"))
		assert.Equal(t, "/foo@bar.org", redactEmail("/foo@bar.org"))
	})
}
