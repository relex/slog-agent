package tparsetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const rfc3339MicroTimestampLayout = "2006-01-02T15:04:05.9Z07:00"

func TestParseRFC3339Timestamp(t *testing.T) {
	timezoneCache := make(map[string]*time.Location)
	for _, timeStr := range []string{
		"2019-08-15T15:50:46.866-08:00",
		"2019-08-15T15:50:46.866915+03:00",
		"2019-08-15T15:50:46.866Z",
	} {
		ourTime, err := parseRFC3339Timestamp(timeStr, timezoneCache)
		assert.Nil(t, err, timeStr+" our parsing")
		theirTime, err := time.Parse(rfc3339MicroTimestampLayout, timeStr)
		assert.Nil(t, err, timeStr+" go parsing")
		assert.Equal(t, theirTime.UnixNano(), ourTime.UnixNano(), timeStr+" UNIX nanoseconds")
		_, ourOffset := ourTime.Zone()
		_, theirOffset := theirTime.Zone()
		assert.Equal(t, theirOffset, ourOffset, timeStr+" TZ offset")
	}
}
