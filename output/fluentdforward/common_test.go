package fluentdforward

import (
	"strings"
	"time"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

var testSchema = base.MustNewLogSchema([]string{"vhost", "app", "message", "extra", "comp"})

var testSerializationConfig = SerializationConfig{
	EnvironmentFields: []string{"vhost", "app"},
	HiddenFields:      []string{"comp"},
	RewriteFields: map[string][]bconfig.LogRewriterConfigHolder{
		"message": {
			{Location: "", Value: &testLogRewriter1{}},
			{Location: "", Value: &testLogRewriter2{}},
		},
	},
}

var testInputRecords = []*base.LogRecord{
	testSchema.NewTestRecord2(time.Date(2010, 12, 1, 10, 30, 40, 50, time.UTC), base.LogFields{"foo", "myapp", "Hello World", "", "Klass1"}),
	testSchema.NewTestRecord2(time.Date(2020, 12, 1, 10, 30, 40, 60, time.UTC), base.LogFields{"bar", "myapp", "Test", "Yes", "Klass2"}),
	testSchema.NewTestRecord2(time.Date(2030, 1, 31, 1, 3, 4, 5, time.Local), base.LogFields{"", "sshd", "account foo logged in", "", "Klass3"}),
	testSchema.NewTestRecord2(time.Date(2035, 2, 28, 11, 33, 44, 55, time.Local), base.LogFields{"", "sshd2", "account bar logged out", "", ""}),
}

var testOutputFieldMaps = []map[string]interface{}{
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

type testLogRewriter1 struct {
	next base.LogRewriter
}

func (rw *testLogRewriter1) GetType() string {
	return "testLogRewriter1"
}

func (rw *testLogRewriter1) NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter {
	rw.next = next
	return rw
}

func (rw *testLogRewriter1) VerifyConfig(schema base.LogSchema, hasNext bool) error {
	return nil
}

func (rw *testLogRewriter1) MaxFieldLength(value string, record *base.LogRecord) int {
	return 65536 + rw.next.MaxFieldLength(value, record)
}

func (rw *testLogRewriter1) WriteFieldBody(value string, record *base.LogRecord, buffer []byte) int {
	n := copy(buffer, "[ ")
	n += rw.next.WriteFieldBody(value, record, buffer[n:])
	n += copy(buffer[n:], " ]")
	return n
}

type testLogRewriter2 struct {
}

func (rw *testLogRewriter2) GetType() string {
	return "testLogRewriter2"
}

func (rw *testLogRewriter2) NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter {
	return rw
}

func (rw *testLogRewriter2) VerifyConfig(schema base.LogSchema, hasNext bool) error {
	return nil
}

func (rw *testLogRewriter2) MaxFieldLength(value string, record *base.LogRecord) int {
	return len(value)
}

func (rw *testLogRewriter2) WriteFieldBody(value string, record *base.LogRecord, buffer []byte) int {
	return copy(buffer, strings.ToUpper(value))
}
