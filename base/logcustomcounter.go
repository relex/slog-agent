package base

import (
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
)

// LogCustomCounterRegistry allows registration of custom record counters by label
//
// RegisterCustomCounter returns a function to be called to count record length
type LogCustomCounterRegistry interface {
	RegisterCustomCounter(label string) func(length int)
}

// logCustomCounterHost hosts log counters by custom labels determined at runtime
// It's used to support for example metrics of different labels added by individual log transforms
type logCustomCounterHost struct {
	countMetricVec  *promext.LazyRWCounterVec
	lengthMetricVec *promext.LazyRWCounterVec
	counterMap      map[string]*logCustomCounter
}

// logCustomCounter represents a pair of (total log count, total log length) metrics by specific label-set
type logCustomCounter struct {
	countMetric     promext.RWCounter
	lengthMetric    promext.RWCounter
	unwrittenCount  uint64
	unwrittenLength uint64
}

// newLogCustomCounterHost creates a logCustomCounterHost bound to a pair of (total log count, total log length) metric vecs
func newLogCustomCounterHost(metricCreator promreg.MetricCreator) *logCustomCounterHost {
	return &logCustomCounterHost{
		countMetricVec:  metricCreator.AddOrGetLazyCounterVec("labelled_records_total", "Numbers of labelled log records", []string{"label"}, nil),
		lengthMetricVec: metricCreator.AddOrGetLazyCounterVec("labelled_record_bytes_total", "Total length in bytes of labelled log records", []string{"label"}, nil),
		counterMap:      make(map[string]*logCustomCounter, 100),
	}
}

// RegisterCustomCounter registers a counter by label and count/length pointers
func (host *logCustomCounterHost) RegisterCustomCounter(label string) func(length int) {
	if counter, exists := host.counterMap[label]; exists {
		return counter.CountRecord
	}
	newCounter := &logCustomCounter{
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

func (cnt *logCustomCounter) CountRecord(recordLength int) {
	cnt.unwrittenCount++
	cnt.unwrittenLength += uint64(recordLength)
}

func (cnt *logCustomCounter) UpdateMetrics() {
	cnt.countMetric.Add(cnt.unwrittenCount)
	cnt.unwrittenCount = 0
	cnt.lengthMetric.Add(cnt.unwrittenLength)
	cnt.unwrittenLength = 0
}
