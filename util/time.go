package util

import (
	"syscall"
	"time"
)

// TimeFromTimeval creates a Time structure from syscall.Timeval
func TimeFromTimeval(val syscall.Timeval) time.Time { // xx:inline
	s, ns := val.Unix()
	return time.Unix(s, ns)
}

// TimeToUnixFloat creates Unix epoch seconds from a Time structure
func TimeToUnixFloat(tm time.Time) float64 { // xx:inline
	return float64(tm.UnixNano()) / 1000000000.0
}
