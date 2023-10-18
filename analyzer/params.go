package analyzer

import (
	"time"
)

const (
	highVolumeDuration      float64 = float64(60 * time.Second)
	highVolumeRecordsPerSec float64 = 10.0 * 1000
	highVolumeBytesPerSec   float64 = 100.0 * 1024 * 1024

	minimalSampleRatio = 0.4
)
