package tparsetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testLayout = "2006-01-02T15:04:05.9Z07:00" // correct layout as required by RFC5424
const testLayoutAlt = "2006-01-02T15:04:05.9Z0700"

type testCase struct {
	timestamp string
	layout    string
}

var testCases = []testCase{
	{"2019-08-15T15:50:46.866-08:00", testLayout},
	{"2019-08-15T15:50:46.866915+03:00", testLayout},
	{"2019-08-15T15:50:46.866Z", testLayout},
	{"2022-02-07T10:30:45.123+0200", testLayoutAlt},
}

func TestParseRFC3339Timestamp(t *testing.T) {
	timezoneCache := make(map[string]*time.Location)
	for _, tc := range testCases {
		ourTime, err := parseRFC3339Timestamp(tc.timestamp, timezoneCache)
		assert.NoError(t, err, tc.timestamp+" our parsing")
		theirTime, err := time.Parse(tc.layout, tc.timestamp)
		assert.NoError(t, err, tc.timestamp+" go parsing")
		assert.Equal(t, theirTime.UnixNano(), ourTime.UnixNano(), tc.timestamp+" UNIX nanoseconds")
		_, ourOffset := ourTime.Zone()
		_, theirOffset := theirTime.Zone()
		assert.Equal(t, theirOffset, ourOffset, tc.timestamp+" TZ offset")
	}
}
