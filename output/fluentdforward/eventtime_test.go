package fluentdforward

import (
	"bytes"
	"testing"
	"time"

	"github.com/relex/fluentlib/protocol/forwardprotocol"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v4"
)

func TestEncodeEventTime(t *testing.T) {
	msg := "test encoding EventTime %s"
	buf := make([]byte, 100)
	start := 0
	end := 0
	decoder := msgpack.NewDecoder(bytes.NewBuffer(buf))
	for _, timeValue := range []time.Time{
		time.Date(1980, 12, 31, 10, 30, 50, 999, time.UTC),
		time.Now(),
		time.Date(2040, 1, 2, 3, 4, 5, 6, time.Local),
	} {
		end = EncodeEventTime(buf, start, timeValue)
		assert.Equal(t, 10, end-start, msg, timeValue)
		tm, derr := decoder.DecodeInterface()
		assert.NoError(t, derr, msg, timeValue)
		assert.Equal(t, timeValue.UnixNano(), tm.(*forwardprotocol.EventTime).Time.UnixNano(), msg, timeValue)
		start = end
	}
}
