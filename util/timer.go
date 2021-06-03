package util

import (
	"time"
)

// ResetTimer resets the given timer properly
func ResetTimer(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(duration)
}
