package test

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/run"
	"github.com/samber/lo"
)

type benchmarkMetric struct {
	fmt string
	val float64
}

// RunBenchmarkPipeline benchmarks a workerless pipeline
func RunBenchmarkPipeline(inputPath string, outputPathPattern string, repeat int, configFile string) {
	mfactory := promreg.NewMetricFactory("benchpipeline_", nil, nil)
	pipeline := preparePipeline(configFile, testOutputTag, mfactory, func(outputName string) chunkSaver {
		return newChunkSaver(outputName, outputPathPattern)
	})

	inputRecords := loadInputRecords(inputPath)
	inputLength := lo.SumBy(inputRecords, func(record []byte) int { return len(record) + 1 /* +1 for newline char */ })

	totalInputCount := len(inputRecords) * repeat
	totalInputLength := int64(inputLength) * int64(repeat)
	costTracker := StartCostTracking()
	pipeline.Run(inputRecords, repeat)

	reportBenchmarkResult(
		"BenchmarkPipeline", totalInputCount, totalInputLength,
		costTracker.Report(), mfactory, pipeline.GetOutputNames(),
	)
	logger.Info(promext.DumpMetrics("", true, false, mfactory))
}

// RunBenchmarkAgent benchmarks a fully configured agent outputting to file or null
func RunBenchmarkAgent(inputPath string, outputPathPattern string, repeat int, configFile string) {
	// launch agent
	loader, confErr := run.NewLoaderFromConfigFile(configFile, "testagent_")
	if confErr != nil {
		logger.Panic(confErr)
	}
	loader.ConfigStats.Log(logger.Root())

	agentInstance := startAgent(loader, createBenchmarkAgentOutputOverride(outputPathPattern), nil, "")

	// feed input
	inputData, numRecords := loadInput(inputPath)
	costTracker := StartCostTracking()
	feedInputToBenchmarkAgent(agentInstance.Address(), inputData, repeat)
	time.Sleep(1 * time.Second)

	logger.Info("stopping...")
	agentInstance.StopAndWait()

	reportBenchmarkResult(
		"BenchmarkAgent", numRecords*repeat, int64(len(inputData))*int64(repeat),
		costTracker.Report(), agentInstance.GetMetricQuerier(), agentInstance.GetOutputNames(),
	)
	logger.Info(promext.DumpMetricsFrom("", true, true, agentInstance.GetMetricQuerier()))
}

func createBenchmarkAgentOutputOverride(outputPathPattern string) base.ChunkConsumerOverrideCreator {
	if outputPathPattern == "" {
		return nil
	}
	return func(parentLogger logger.Logger, name string, decoder base.ChunkDecoder, args base.ChunkConsumerArgs) base.ChunkConsumer {
		return newChunkSavingWorker(parentLogger, decoder, args, newChunkSaver(name, outputPathPattern))
	}
}

func feedInputToBenchmarkAgent(agentAddress string, inputData []byte, repeat int) {
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

func reportBenchmarkResult(
	title string, numLogs int, sizeOfLogs int64,
	report CostReport, mquerier promreg.MetricQuerier, outputNames []string,
) {
	printBenchmarkMetrics(title, []benchmarkMetric{
		{fmt: "%.0f log/sec", val: float64(numLogs) / report.RealTime.Seconds()},
		{fmt: "%.0f MB/sec", val: float64(sizeOfLogs) / 1048576.0 / report.RealTime.Seconds()},
		{fmt: "%0.2f alloc/log", val: float64(report.NumHeapAllocs) / float64(numLogs)},
		{fmt: "%0.2f%% user", val: 100.0 * report.UserTime.Seconds() / report.RealTime.Seconds()},
		{fmt: "%0.2f%% sys", val: 100.0 * report.SystemTime.Seconds() / report.RealTime.Seconds()},
		{fmt: "%0.2f%% gc", val: 100.0 * report.GCCPUFraction},
		{fmt: "%.02f sec", val: report.RealTime.Seconds()},
	})

	numPass := promext.SumMetricValues(mquerier.LookupMetricFamily("process_passed_records_total"))
	numDrop := promext.SumMetricValues(mquerier.LookupMetricFamily("process_dropped_records_total"))
	if int(numPass)+int(numDrop) != numLogs {
		logger.Errorf("numbers of processed records don't match: %d, should be %d", int(numPass)+int(numDrop), numLogs)
	}

	printBenchmarkMetrics(title+":filter", []benchmarkMetric{
		{fmt: "%.0f logs passed", val: numPass},
		{fmt: "%.0f logs dropped", val: numDrop},
		{fmt: "%.0f MB in", val: float64(sizeOfLogs) / 1048576},
	})

	for _, outName := range outputNames {
		labels := prometheus.Labels{"output": outName}

		encBytes := promext.SumMetricValues2(mquerier.LookupMetricFamily("process_serialized_bytes_total"), labels)
		outChunks := promext.SumMetricValues2(mquerier.LookupMetricFamily("process_chunks_total"), labels)
		outBytes := promext.SumMetricValues2(mquerier.LookupMetricFamily("process_chunk_bytes_total"), labels)

		printBenchmarkMetrics(title+":output:"+outName, []benchmarkMetric{
			{fmt: "%.0f MB encoded", val: float64(encBytes) / 1048576.0},
			{fmt: "%.0f log/chunk", val: float64(numLogs) / outChunks},
			{fmt: "%.0f KB/chunk", val: float64(outBytes) / 1024.0 / outChunks},
			{fmt: "%.0f MB out", val: float64(outBytes) / 1048576.0},
		})
	}
}

func printBenchmarkMetrics(title string, metrics []benchmarkMetric) {
	sb := make([]byte, 0, 200)
	sb = append(sb, fmt.Sprintf("%s:", title)...)
	for _, m := range metrics {
		sb = append(sb, fmt.Sprintf("\t"+m.fmt, m.val)...)
	}
	fmt.Println(string(sb))
}
