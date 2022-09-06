package base

import (
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/util"
)

// LogProcessCounter tracks metrics for log transform, serialization and chunk making
//
// LogProcessCounter must be accessed through pointer. It's not concurrently usable. Counter-vectors and counters
// created here may duplicate with others, as long as the labels match.
//
// It tracks counters by metric keys (ex: vhost+source) that are not part of orchestration keys (ex: level), by
// creating a fixed-length array containing counters for each of key-set. The positions of final counters are decided
// during registration process.
//
// LogInputCounter's own custom counter registry is ignored here, as map access per counter update would be very
// inefficient.
type LogProcessCounter struct {
	factory                promreg.MetricCreator
	metricKeyExtractor     FieldSetExtractor                     // to extract metric keys from log records
	metricKeyNames         []string                              // label names of metric keys (ex: key_vhost)
	customCounterVecMap    map[string]logProcessCustomCounterVec // map of custom label => counter-vector[label], with unfilled metric key labels
	inputCounterPairByKeys map[string]logInputCounterPair        // map of merged metric key => (input counter, custom counters)
	currentCustomCounters  []*logCustomCounter                   // currently selected counter array of specific key-set
	serializedLengthTotal  []valueCounterProvider                // an array of per-output metrics counters, accessed by output index
	chunksCountTotal       []valueCounterProvider
	chunksLengthTotal      []valueCounterProvider
	mergeKeyBuffer         []byte // reused buffer to build merged metric key from record
}

type logProcessCustomCounterVec struct {
	index           int
	countMetricVec  *promext.LazyRWCounterVec
	lengthMetricVec *promext.LazyRWCounterVec
}

type logInputCounterPair struct {
	inputCounter   *LogInputCounter
	customCounters []*logCustomCounter
}

// NewLogProcessCounter creates a LogProcessCounter
func NewLogProcessCounter(factory promreg.MetricCreator, schema LogSchema, keyLocators []LogFieldLocator, outputNames []string) *LogProcessCounter {
	metricKeyNames := make([]string, len(keyLocators))
	for i, loc := range keyLocators {
		metricKeyNames[i] = "key_" + loc.Name(schema)
	}
	counter := &LogProcessCounter{
		factory:                factory,
		metricKeyExtractor:     *NewFieldSetExtractor(keyLocators),
		metricKeyNames:         metricKeyNames,
		customCounterVecMap:    make(map[string]logProcessCustomCounterVec, 100),
		inputCounterPairByKeys: make(map[string]logInputCounterPair, 2000),
		currentCustomCounters:  nil,
		mergeKeyBuffer:         make([]byte, 0, 200),
	}

	counter.serializedLengthTotal = make([]valueCounterProvider, len(outputNames))
	counter.chunksCountTotal = make([]valueCounterProvider, len(outputNames))
	counter.chunksLengthTotal = make([]valueCounterProvider, len(outputNames))

	for i, output := range outputNames {
		counter.serializedLengthTotal[i] = valueCounterProvider{
			factory.AddOrGetCounter("serialized_bytes_total", "Total lengths in bytes of serialized log records", []string{"output"}, []string{output}), 0,
		}
		counter.chunksCountTotal[i] = valueCounterProvider{
			factory.AddOrGetCounter("chunks_total", "Numbers of created chunks", []string{"output"}, []string{output}), 0,
		}
		counter.chunksLengthTotal[i] = valueCounterProvider{
			factory.AddOrGetCounter("chunk_bytes_total", "Total length in bytes of created chunks", []string{"output"}, []string{output}), 0,
		}
	}

	return counter
}

// RegisterCustomCounter registers a custom counter by label and count/length pointers
//
// This method must not be called in processing stage, when counters are already being selected and updated
func (pcounter *LogProcessCounter) RegisterCustomCounter(label string) func(length int) {
	counterVec, exists := pcounter.customCounterVecMap[label]
	if !exists {
		counterVec = logProcessCustomCounterVec{
			index: len(pcounter.customCounterVecMap),
			countMetricVec: pcounter.factory.AddOrGetLazyCounterVec("labelled_records_total", "Numbers of labelled log records",
				append([]string{"label"}, pcounter.metricKeyNames...), []string{label}),
			lengthMetricVec: pcounter.factory.AddOrGetLazyCounterVec("labelled_record_bytes_total", "Total length in bytes of labelled log records",
				append([]string{"label"}, pcounter.metricKeyNames...), []string{label}),
		}
		pcounter.customCounterVecMap[label] = counterVec
	}
	counterVecIndex := counterVec.index
	return func(length int) {
		c := pcounter.currentCustomCounters[counterVecIndex]
		c.unwrittenCount++
		c.unwrittenLength += uint64(length)
	}
}

// SelectInputCounter gets or adds the input counter for key-set of the given record and marks its custom counter
// array as current, for subsequent transforms to update counter values for the correct key-set
func (pcounter *LogProcessCounter) SelectInputCounter(record *LogRecord) *LogInputCounter {
	tempKeys := pcounter.metricKeyExtractor.Extract(record)

	tempMergedKey := pcounter.mergeKeyBuffer
	for _, tkey := range tempKeys {
		tempMergedKey = append(tempMergedKey, tkey...)
	}
	pcounter.mergeKeyBuffer = tempMergedKey[:0]

	// try to get existing counter by temp key, no new key string is created here
	pair, found := pcounter.inputCounterPairByKeys[string(tempMergedKey)]
	if !found {
		// copy transient field values from record for storing into map and counters
		permKeys := util.DeepCopyStrings(tempKeys)
		permMergedKey := util.DeepCopyStringFromBytes(tempMergedKey)
		customCounters := make([]*logCustomCounter, len(pcounter.customCounterVecMap))
		for _, vec := range pcounter.customCounterVecMap {
			customCounters[vec.index] = &logCustomCounter{
				countMetric:     vec.countMetricVec.WithLabelValues(permKeys...),
				lengthMetric:    vec.lengthMetricVec.WithLabelValues(permKeys...),
				unwrittenCount:  0,
				unwrittenLength: 0,
			}
		}
		pair = logInputCounterPair{
			inputCounter:   NewLogInputCounter(pcounter.factory.AddOrGetPrefix("", pcounter.metricKeyNames, permKeys)),
			customCounters: customCounters,
		}
		pcounter.inputCounterPairByKeys[permMergedKey] = pair
	}

	pcounter.currentCustomCounters = pair.customCounters
	return pair.inputCounter
}

// CountStream updates counters for stream serialization
func (pcounter *LogProcessCounter) CountStream(outputIndex int, stream LogStream) { // xx:inline
	pcounter.serializedLengthTotal[outputIndex].unwrittenValue += uint64(len(stream))
}

// CountChunk updates counters for chunk generation
func (pcounter *LogProcessCounter) CountChunk(outputIndex int, chunk *LogChunk) { // xx:inline
	pcounter.chunksCountTotal[outputIndex].unwrittenValue++
	pcounter.chunksLengthTotal[outputIndex].unwrittenValue += uint64(len(chunk.Data))
}

// UpdateMetrics writes unwritten values in the counter to underlying Prometheus counters
func (pcounter *LogProcessCounter) UpdateMetrics() {
	for _, pair := range pcounter.inputCounterPairByKeys {
		pair.inputCounter.UpdateMetrics()
		for _, counter := range pair.customCounters {
			counter.UpdateMetrics()
		}
	}

	// all these slices should have the same length, so we can iterate over them in one loop
	for i := range pcounter.serializedLengthTotal {
		pcounter.serializedLengthTotal[i].UpdateMetric()
		pcounter.chunksCountTotal[i].UpdateMetric()
		pcounter.chunksLengthTotal[i].UpdateMetric()
	}
}
