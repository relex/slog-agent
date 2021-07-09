package fluentdforward

import (
	"time"

	"github.com/relex/slog-agent/output/fastmsgpack"
)

// EncodeEventTime encodes a float64 as EventTime
func EncodeEventTime(buffer []byte, start int, value time.Time) int {
	pos := fastmsgpack.EncodeExtHeader8(buffer, start, 0)
	pos = fastmsgpack.Write4(buffer, pos, uint32(value.Unix()))
	return fastmsgpack.Write4(buffer, pos, uint32(value.Nanosecond()))
}
