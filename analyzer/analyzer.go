package analyzer

import (
	"time"

	"github.com/relex/slog-agent/base"
)

type simpleAnalyzer struct {
	tickTraffic       tickTrafficInfo
	highVolumeStart   time.Time
	analysisScheduled bool
}

type tickTrafficInfo struct {
	startTime  time.Time
	numRecords int
	numBytes   int64
}

func NewAnalyzer() base.LogAnalyzer {
	return &simpleAnalyzer{
		tickTraffic: tickTrafficInfo{
			startTime:  time.Now(),
			numRecords: 0,
			numBytes:   0,
		},
		highVolumeStart:   time.Time{},
		analysisScheduled: false,
	}
}

func (a *simpleAnalyzer) ShouldAnalyze(batch base.LogRecordBatch) bool {
	// we cannot analyze any batch that is not full, as there would be too few samples.
	return a.analysisScheduled && batch.Full
}

func (a *simpleAnalyzer) TrackTraffic(numCleanRecords int, numCleanBytes int64) {
	a.tickTraffic.numRecords += numCleanRecords
	a.tickTraffic.numBytes += numCleanBytes
}

func (a *simpleAnalyzer) Analyze(batch base.LogRecordBatch, numCleanRecords int, numCleanBytes int64) {
	if float64(numCleanRecords)/float64(len(batch.Records)) < minimalSampleRatio &&
		float64(numCleanBytes)/float64(batch.NumBytes) < minimalSampleRatio {
		return
	}

	a.highVolumeStart = time.Time{}
	a.analysisScheduled = false

	// TODO: analysis
}

func (a *simpleAnalyzer) Tick() {
	now := time.Now()
	durationSec := float64(now.Sub(a.tickTraffic.startTime)+1) / float64(time.Second)
	highVolume := float64(a.tickTraffic.numRecords)/durationSec >= highVolumeRecordsPerSec ||
		float64(a.tickTraffic.numBytes)/durationSec >= highVolumeBytesPerSec

	if highVolume {
		if a.highVolumeStart.IsZero() {
			a.highVolumeStart = now
			a.analysisScheduled = false
		} else if float64(now.Sub(a.highVolumeStart))/float64(time.Second) >= highVolumeDuration {
			a.analysisScheduled = true
		}
	} else {
		a.highVolumeStart = time.Time{}
		a.analysisScheduled = false // cancel any if scheduled but not done
	}

	a.tickTraffic = tickTrafficInfo{
		startTime:  now,
		numRecords: 0,
		numBytes:   0,
	}
}
