package base

import (
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
)

// logCustomCounterHost hosts log counters by custom labels determined at runtime.
//
// It's used to support for example metrics of different labels added by individual log transforms.
type logCustomCounterHost struct {
	countMetricVec  *promext.LazyRWCounterVec // Use lazy-init counters as there could be many unused metrics for many pipelines
	lengthMetricVec *promext.LazyRWCounterVec
	counterMap      map[string]*logCustomCounterImpl
}

// logCustomCounterImpl represents a pair of (total log count, total log length) metrics by specific label-set
type logCustomCounterImpl struct {
	countMetric     promext.LazyRWCounter
	lengthMetric    promext.LazyRWCounter
	unwrittenCount  uint64
	unwrittenLength uint64
}

// newLogCustomCounterHost creates a logCustomCounterHost bound to a pair of (total log count, total log length) metric vecs
func newLogCustomCounterHost(metricCreator promreg.MetricCreator) *logCustomCounterHost {
	return &logCustomCounterHost{
		countMetricVec:  metricCreator.AddOrGetLazyCounterVec("labelled_records_total", "Numbers of labelled log records", []string{"label"}, nil),
		lengthMetricVec: metricCreator.AddOrGetLazyCounterVec("labelled_record_bytes_total", "Total length in bytes of labelled log records", []string{"label"}, nil),
		counterMap:      make(map[string]*logCustomCounterImpl, 100),
	}
}

// RegisterCustomCounter registers a counter by label and count/length pointers
func (host *logCustomCounterHost) RegisterCustomCounter(label string) func(length int) {
	if counter, exists := host.counterMap[label]; exists {
		return counter.CountRecord
	}
	newCounter := &logCustomCounterImpl{
		countMetric:     host.countMetricVec.WithLabelValues(label),
		lengthMetric:    host.lengthMetricVec.WithLabelValues(label),
		unwrittenCount:  0,
		unwrittenLength: 0,
	}
	host.counterMap[label] = newCounter
	return newCounter.CountRecord
}

// UpdateMetrics writes values from counter providers to underlying Prometheus counters
func (host *logCustomCounterHost) UpdateMetrics() {
	for _, counter := range host.counterMap {
		counter.UpdateMetrics()
	}
}

func (cnt *logCustomCounterImpl) CountRecord(recordLength int) {
	cnt.unwrittenCount++
	cnt.unwrittenLength += uint64(recordLength)
}

func (cnt *logCustomCounterImpl) UpdateMetrics() {
	cnt.countMetric.Add(cnt.unwrittenCount)
	cnt.unwrittenCount = 0
	cnt.lengthMetric.Add(cnt.unwrittenLength)
	cnt.unwrittenLength = 0
}
