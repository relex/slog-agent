package base

type LogAnalyzer interface {
	ShouldAnalyze(batch LogRecordBatch) bool

	TrackTraffic(numCleanRecords int, numCleanBytes int64)
	Analyze(batch LogRecordBatch, numCleanRecords int, numCleanBytes int64)

	Tick()
}
