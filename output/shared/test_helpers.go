package shared

import (
	"strings"
	"time"

	"github.com/relex/slog-agent/base"
)

var TestSchema = base.MustNewLogSchema([]string{"vhost", "app", "message", "extra", "comp"})

var TestInputRecords = []*base.LogRecord{
	TestSchema.NewTestRecord2(time.Date(2010, 12, 1, 10, 30, 40, 50, time.UTC), base.LogFields{"foo", "myapp", "Hello World", "", "Klass1"}),
	TestSchema.NewTestRecord2(time.Date(2020, 12, 1, 10, 30, 40, 60, time.UTC), base.LogFields{"bar", "myapp", "Test", "Yes", "Klass2"}),
	TestSchema.NewTestRecord2(time.Date(2030, 1, 31, 1, 3, 4, 5, time.Local), base.LogFields{"", "sshd", "account foo logged in", "", "Klass3"}),
	TestSchema.NewTestRecord2(time.Date(2035, 2, 28, 11, 33, 44, 55, time.Local), base.LogFields{"", "sshd2", "account bar logged out", "", ""}),
}

var TestOutputFieldMaps = []map[string]interface{}{
	{
		"message": "[ HELLO WORLD ]",
		"environment": map[string]interface{}{
			"vhost": "foo",
			"app":   "myapp",
		},
	},
	{
		"message": "[ TEST ]",
		"extra":   "Yes",
		"environment": map[string]interface{}{
			"vhost": "bar",
			"app":   "myapp",
		},
	},
	{
		"message": "[ ACCOUNT FOO LOGGED IN ]",
		"environment": map[string]interface{}{
			"vhost": "",
			"app":   "sshd",
		},
	},
	{
		"message": "[ ACCOUNT BAR LOGGED OUT ]",
		"environment": map[string]interface{}{
			"vhost": "",
			"app":   "sshd2",
		},
	},
}

type TestLogRewriter1 struct {
	next base.LogRewriter
}

func (rw *TestLogRewriter1) GetType() string {
	return "testLogRewriter1"
}

func (rw *TestLogRewriter1) NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter {
	rw.next = next
	return rw
}

func (rw *TestLogRewriter1) VerifyConfig(schema base.LogSchema, hasNext bool) error {
	return nil
}

func (rw *TestLogRewriter1) MaxFieldLength(value string, record *base.LogRecord) int {
	return 65536 + rw.next.MaxFieldLength(value, record)
}

func (rw *TestLogRewriter1) WriteFieldBody(value string, record *base.LogRecord, buffer []byte) int {
	n := copy(buffer, "[ ")
	n += rw.next.WriteFieldBody(value, record, buffer[n:])
	n += copy(buffer[n:], " ]")
	return n
}

type TestLogRewriter2 struct{}

func (rw *TestLogRewriter2) GetType() string {
	return "testLogRewriter2"
}

func (rw *TestLogRewriter2) NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter {
	return rw
}

func (rw *TestLogRewriter2) VerifyConfig(schema base.LogSchema, hasNext bool) error {
	return nil
}

func (rw *TestLogRewriter2) MaxFieldLength(value string, record *base.LogRecord) int {
	return len(value)
}

func (rw *TestLogRewriter2) WriteFieldBody(value string, record *base.LogRecord, buffer []byte) int {
	return copy(buffer, strings.ToUpper(value))
}
