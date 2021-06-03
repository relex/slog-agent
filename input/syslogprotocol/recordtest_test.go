package syslogprotocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyslogMultiLineStart(t *testing.T) {
	assert.False(t, TestRecordStart([]byte("something")))
	assert.False(t, TestRecordStart([]byte("<something aaaaa bbbbb ccccc ddddd eeeee fffff")))
	assert.True(t, TestRecordStart([]byte("<1>1 2019-08-15T15:50:46.866915+03:00 local1 my-app1 123 fn1 - Something")))
	assert.True(t, TestRecordStart([]byte("<16>1 2019-08-15T15:50:46.866915+03:00 local1 my-app1 123 fn1 - Something")))
	assert.True(t, TestRecordStart([]byte("<163>1 2019-08-15T15:50:46.866915+03:00 local1 my-app1 123 fn1 - Something")))
	assert.False(t, TestRecordStart([]byte("<1634>1 2019-08-15T15:50:46.866915+03:00 local1 my-app1 123 fn1 - Something")))
}
