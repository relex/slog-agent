package test

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/util"
)

type benchmarkMetric struct {
	fmt string
	val float64
}

// RunBenchmarkPipeline benchmarks a workerless pipeline
func RunBenchmarkPipeline(inputPath string, outputPath string, repeat int, configFile string) {
	mfactory := base.NewMetricFactory("benchpipeline_", nil, nil)
	outputConfig, process, endProcess := preparePipeline(configFile, testOutputTag, mfactory)
	writeChunk, closeOutput := openLogChunkConsumingFunc(outputPath, outputConfig)

	inputRecords := loadInputRecords(inputPath)
	inputLength := 0
	util.Each(len(inputRecords), func(i int) { inputLength += len(inputRecords[i]) + 1 })

	totalInputCount := len(inputRecords) * repeat
	totalInputLength := int64(inputLength) * int64(repeat)
	costTracker := StartCostTracking()
	runPipeline(process, endProcess, inputRecords, repeat, writeChunk)
	closeOutput()

	reportBenchmarkResult("BenchmarkPipeline", totalInputCount, totalInputLength, costTracker.Report(), mfactory)
	logger.Info(mfactory.DumpMetrics(false))
}

// RunBenchmarkAgent benchmarks a fully configured agent outputting to file or null
func RunBenchmarkAgent(inputPath string, outputPath string, repeat int, configFile string) {
	// launch agent
	loader, loaderErr := run.NewLoaderFromConfigFile(configFile, "testagent_")
	if loaderErr != nil {
		logger.Panic(loaderErr)
	}

	chunkSaver := openLogChunkSaver(outputPath, loader.Output)
	agt := startAgent(loader, chunkSaver, nil, "")

	// feed input
	inputData, numRecords := loadInput(inputPath)
	costTracker := StartCostTracking()
	runBenchmarkInputSender(agt.Address(), inputData, repeat)
	time.Sleep(1 * time.Second)

	logger.Info("stopping...")
	agt.StopAndWait()

	reportBenchmarkResult("BenchmarkAgent", numRecords*repeat, int64(len(inputData))*int64(repeat), costTracker.Report(), agt.MetricFactory())
	logger.Info(agt.DumpMetrics())
}

func runBenchmarkInputSender(agentAddress string, inputData []byte, repeat int) {
	const minFrameSize = 1 * 1024 * 1024
	const maxFrameSize = 1 * 1024 * 1024

	runtime.LockOSThread()

	conn, err := net.Dial("tcp", agentAddress)
	if err != nil {
		logger.Fatal("connect: ", err.Error())
	}

	numSent := int64(0)
	if len(inputData) >= minFrameSize {
		for i := 0; i < repeat; i++ {
			n, err := conn.Write(inputData)
			if err != nil {
				logger.Fatal("error sending ", err.Error())
			}
			numSent += int64(n)
		}
	} else {
		normalFrameRepeat := maxFrameSize/len(inputData) + 1
		normalFrame := make([]byte, len(inputData)*normalFrameRepeat)
		{
			offset := 0
			for i := 0; i < normalFrameRepeat; i++ {
				offset += copy(normalFrame[offset:], inputData)
			}
		}

		lastFrameRepeat := repeat % normalFrameRepeat
		lastFrame := normalFrame[:len(inputData)*lastFrameRepeat]
		for i := 0; i < repeat/normalFrameRepeat; i++ {
			n, err := conn.Write(normalFrame)
			if err != nil {
				logger.Fatal("error sending: ", err.Error())
			}
			numSent += int64(n)
		}
		if n, err := conn.Write(lastFrame); err != nil {
			logger.Fatal("error sending last: ", err.Error())
		} else {
			numSent += int64(n)
		}
	}

	if err := conn.Close(); err != nil {
		logger.Fatal("close: ", err.Error())
	}
	logger.Infof("writer sent %d bytes", numSent)

	runtime.UnlockOSThread()
}

func reportBenchmarkResult(title string, numLogs int, sizeOfLogs int64, report CostReport, mfactory *base.MetricFactory) {
	metrics := []benchmarkMetric{
		{fmt: "%.0f log/sec", val: float64(numLogs) / report.RealTime.Seconds()},
		{fmt: "%.0f MB/sec", val: float64(sizeOfLogs) / 1048576 / report.RealTime.Seconds()},
		{fmt: "%0.2f alloc/log", val: float64(report.NumHeapAllocs) / float64(numLogs)},
		{fmt: "%0.2f%% user", val: 100.0 * report.UserTime.Seconds() / report.RealTime.Seconds()},
		{fmt: "%0.2f%% sys", val: 100.0 * report.SystemTime.Seconds() / report.RealTime.Seconds()},
		{fmt: "%0.2f%% gc", val: 100.0 * report.GCCPUFraction},
		{fmt: "%.02f sec", val: report.RealTime.Seconds()},
	}
	numPass := util.SumMetricValues(mfactory.AddOrGetCounterVec("process_passed_records_total", "", nil, nil))
	numDrop := util.SumMetricValues(mfactory.AddOrGetCounterVec("process_dropped_records_total", "", nil, nil))
	if int(numPass)+int(numDrop) != numLogs {
		logger.Errorf("numbers of processed records don't match: %d, should be %d", int(numPass)+int(numDrop), numLogs)
	}
	numChunks := util.SumMetricValues(mfactory.AddOrGetCounterVec("process_chunks_total", "", nil, nil))
	numBytes := util.SumMetricValues(mfactory.AddOrGetCounterVec("process_chunk_bytes_total", "", nil, nil))
	metrics = append(metrics, benchmarkMetric{fmt: "%.0f log/chunk", val: float64(numLogs) / numChunks})
	metrics = append(metrics, benchmarkMetric{fmt: "%.0f KB/chunk", val: numBytes / 1024.0 / numChunks})
	metrics = append(metrics, benchmarkMetric{fmt: "%.0f MB in", val: float64(sizeOfLogs) / 1048576})
	metrics = append(metrics, benchmarkMetric{fmt: "%.0f MB out", val: numBytes / 1048576})
	printBenchmarkMetrics(title, metrics)
}

func printBenchmarkMetrics(title string, metrics []benchmarkMetric) {
	sb := make([]byte, 0, 200)
	sb = append(sb, fmt.Sprintf("%s:", title)...)
	for _, m := range metrics {
		sb = append(sb, fmt.Sprintf("\t"+m.fmt, m.val)...)
	}
	fmt.Println(string(sb))
}
