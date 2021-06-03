package base

import "github.com/relex/gotils/promexporter"

// LogInputCounter tracks metrics for incoming logs
//
// LogInputCounter must be accessed through pointer. It's not concurrently usable. Counter-vectors and counters
// created here may duplicate with others, as long as the labels match.
type LogInputCounter struct {
	logCustomCounterHost
	passedRecordsCountTotal   valueCounterProvider
	passedRecordsLengthTotal  valueCounterProvider
	droppedRecordsCountTotal  valueCounterProvider
	droppedRecordsLengthTotal valueCounterProvider
}

// valueCounter provides a counter metric
type valueCounterProvider struct {
	metric         promexporter.RWCounter // metric for accumulated count of something
	unwrittenValue uint64                 // accumulated count of something not yet written to Prometheus metrics
}

// NewLogInputCounter creates a LogInputCounter
func NewLogInputCounter(factory *MetricFactory) *LogInputCounter {
	return &LogInputCounter{
		logCustomCounterHost: *newLogCustomCounterHost(factory),
		passedRecordsCountTotal: valueCounterProvider{
			factory.AddOrGetCounter("passed_records_total", "Numbers of passed log records", nil, nil), 0,
		},
		passedRecordsLengthTotal: valueCounterProvider{
			factory.AddOrGetCounter("passed_record_bytes_total", "Total length in bytes of passed log records", nil, nil), 0,
		},
		droppedRecordsCountTotal: valueCounterProvider{
			factory.AddOrGetCounter("dropped_records_total", "Numbers of dropped log records", nil, nil), 0,
		},
		droppedRecordsLengthTotal: valueCounterProvider{
			factory.AddOrGetCounter("dropped_record_bytes_total", "Total length in bytes of dropped log records", nil, nil), 0,
		},
	}
}

// CountRecordPass updates counters for log record passing
func (icounter *LogInputCounter) CountRecordPass(record *LogRecord) { // xx:inline
	icounter.passedRecordsCountTotal.unwrittenValue++
	icounter.passedRecordsLengthTotal.unwrittenValue += uint64(record.RawLength)
}

// CountRecordDrop updates counters for log record dropping
func (icounter *LogInputCounter) CountRecordDrop(record *LogRecord) { // xx:inline
	icounter.droppedRecordsCountTotal.unwrittenValue++
	icounter.droppedRecordsLengthTotal.unwrittenValue += uint64(record.RawLength)
}

// UpdateMetrics writes unwritten values in the counter to underlying Prometheus counters
func (icounter *LogInputCounter) UpdateMetrics() {
	icounter.logCustomCounterHost.UpdateMetrics()

	icounter.passedRecordsCountTotal.UpdateMetric()
	icounter.passedRecordsLengthTotal.UpdateMetric()
	icounter.droppedRecordsCountTotal.UpdateMetric()
	icounter.droppedRecordsLengthTotal.UpdateMetric()
}

func (vprov *valueCounterProvider) UpdateMetric() {
	vprov.metric.Add(vprov.unwrittenValue)
	vprov.unwrittenValue = 0
}
