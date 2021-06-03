package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

const TestOutputGenerationArg = "gen"

func TestDataGeneration(t *testing.T) {
	if util.IndexOfString(os.Args, TestOutputGenerationArg) == -1 {
		return
	}
	t.Log("regenerate log outputs...")
	mfactory := base.NewMetricFactory("testpipeline_", nil, nil)
	inputFiles, err := filepath.Glob("../testdata/development/*-input.log")
	if err != nil {
		logger.Fatal(err)
	}
	for _, inPath := range inputFiles {
		inLines := loadInputRecords(inPath)
		title := filepath.Base(inPath[:len(inPath)-10])
		outConfig, process, endProcess := preparePipeline("../testdata/config_sample.yml", title, mfactory)
		outPath := fmt.Sprintf("../testdata/development/%s-output.json", title)
		outWrite, outClose := openLogChunkConsumingFunc(outPath, outConfig)
		runPipeline(process, endProcess, inLines, 1, outWrite)
		outClose()
	}
	if dump, derr := mfactory.DumpMetrics(false); assert.Nil(t, derr) {
		assert.Nil(t, ioutil.WriteFile("../testdata/development/all-pipeline.prom", []byte(dump), 0644))
	}
}

func TestPipeline(t *testing.T) {
	if util.IndexOfString(os.Args, TestOutputGenerationArg) != -1 {
		return
	}
	mfactory := base.NewMetricFactory("testpipeline_", nil, nil)
	inputFiles, err := filepath.Glob("../testdata/development/*-input.log")
	if err != nil {
		assert.Nil(t, err)
	}
	if len(inputFiles) == 0 {
		assert.NotZero(t, len(inputFiles))
	}
	for _, inPath := range inputFiles {
		localInPath := inPath
		title := filepath.Base(inPath[:len(inPath)-10])
		expectedOutPath := fmt.Sprintf("../testdata/development/%s-output.json", title)
		t.Run(title, func(tt *testing.T) {
			expectedOutput, eOutErr := ioutil.ReadFile(expectedOutPath)
			assert.Nil(t, eOutErr)
			outConfig, process, endProcess := preparePipeline("../testdata/config_sample.yml", title, mfactory)
			outWrite, outClose, outGet := openJSONMemWriter(outConfig)
			runPipeline(process, endProcess, loadInputRecords(localInPath), 1, outWrite)
			outClose()
			assert.Equal(t, string(expectedOutput), outGet())
		})
	}
	t.Run("check metrics", func(tt *testing.T) {
		expectedMetrics, err := ioutil.ReadFile("../testdata/development/all-pipeline.prom")
		if assert.Nil(tt, err) {
			if dump, derr := mfactory.DumpMetrics(false); assert.Nil(tt, derr) {
				assert.Equal(tt, string(expectedMetrics), dump)
			}
		}
	})
}

func TestAgent(t *testing.T) {
	regenOutput := util.IndexOfString(os.Args, TestOutputGenerationArg) != -1
	inputDataByTag, expectedResultsByTag := buildInputsAndExpectedOutputs(t, "../testdata/development")
	config, schema, configErr := run.LoadConfigFile("../testdata/config_sample.yml")
	assert.Nil(t, configErr)
	outputWritersByTag, newChunkSaver := prepareOutputWriters(t, config.Output.LogOutputConfig)
	// Override tag for output splitting and keys (labelsets) for distribution: order of logs would be messed up if keys are different
	ag, aerr := startAgent(config, schema, newChunkSaver, []string{"host"}, "$host")
	assert.Nil(t, aerr)
	// Write input
	inputWorkerCounter := &sync.WaitGroup{}
	for tag, input := range inputDataByTag {
		logger.Infof("launching writer for input tag=%s len=%d", tag, len(input))
		inputWorkerCounter.Add(1)
		go func(input []byte) {
			runBenchmarkInputSender(ag.Address(), input, 1)
			inputWorkerCounter.Done()
		}(input)
	}
	// Wait for ending
	logger.Infof("waiting for %d input writers...", len(inputDataByTag))
	inputWorkerCounter.Wait()
	time.Sleep(1 * time.Second) // TODO: find out why socket accept, receive and allow conn close before go code is invoked?
	logger.Info("stopping agent...")
	ag.StopAndWait()
	finalizeOutputWriters(t, outputWritersByTag)
	// compare
	if !regenOutput {
		t.Run("check known outputs", func(tt *testing.T) {
			for tag, str := range expectedResultsByTag {
				wrt, ok := outputWritersByTag[tag]
				if assert.True(tt, ok, tag) {
					assert.Equal(tt, str, wrt.String())
				}
			}
		})
		t.Run("check unexpected outputs", func(tt *testing.T) {
			for tag := range outputWritersByTag {
				_, ok := expectedResultsByTag[tag]
				assert.True(tt, ok, tag)
			}
		})
		t.Run("check metrics", func(tt *testing.T) {
			expectedMetrics, err := ioutil.ReadFile("../testdata/development/all-agent.prom")
			if assert.Nil(tt, err) {
				assert.Equal(tt, string(expectedMetrics), ag.DumpMetrics())
			}
		})
	} else {
		// output JSONs are to be generated from pipeline test, not here
		assert.Nil(t, ioutil.WriteFile("../testdata/development/all-agent.prom", []byte(ag.DumpMetrics()), 0644))
	}
}

func buildInputsAndExpectedOutputs(t *testing.T, baseDir string) (map[string][]byte, map[string]string) {
	inputFiles, err := filepath.Glob(baseDir + "/*-input.log")
	if err != nil {
		assert.Nil(t, err)
	}
	if len(inputFiles) == 0 {
		assert.NotZero(t, len(inputFiles))
	}
	inputDataByTag := make(map[string][]byte, len(inputFiles))
	expectedResultsByTag := make(map[string]string, len(inputFiles))
	for _, inPath := range inputFiles {
		title := filepath.Base(inPath[:len(inPath)-len("-input.log")])
		{
			inputData, ierr := ioutil.ReadFile(inPath)
			assert.Nil(t, ierr)
			inputDataByTag[title] = inputData
		}
		{
			expectedOutPath := fmt.Sprintf(baseDir+"/%s-output.json", title)
			expectedResult, oerr := ioutil.ReadFile(expectedOutPath)
			assert.Nil(t, oerr)
			expectedResultsByTag[title] = string(expectedResult)
		}
	}
	return inputDataByTag, expectedResultsByTag
}

func prepareOutputWriters(t *testing.T, outputConfig bconfig.LogOutputConfig) (map[string]*bytes.Buffer, base.ChunkConsumerConstructor) {
	outputWritersByTag := make(map[string]*bytes.Buffer, 100)
	getOutputWriterByTag := func(tag string) *bytes.Buffer {
		wrt, exists := outputWritersByTag[tag]
		if !exists {
			wrt = bytes.NewBuffer(make([]byte, 0, 1048576))
			outputWritersByTag[tag] = wrt
		}
		return wrt
	}
	mutex := &sync.Mutex{}
	collectChunkJSON := func(chunk base.LogChunk) {
		mutex.Lock()
		defer mutex.Unlock()
		buf := bytes.Buffer{}
		info, derr := outputConfig.DumpRecordsAsJSON(chunk, []byte(",\n"), true, &buf)
		if derr != nil {
			t.Errorf("failed to decode chunk: %s", derr.Error())
			return
		}
		logger.Infof("chunk: tag=%s count=%d", info.Tag, info.NumRecords)
		wrt := getOutputWriterByTag(info.Tag)
		if wrt.Len() == 0 {
			if _, err := wrt.Write([]byte("[\n")); err != nil {
				t.Errorf("failed to write separator: %w", err)
				return
			}
		} else {
			if _, err := wrt.Write([]byte(",\n")); err != nil {
				t.Errorf("failed to write separator: %w", err)
				return
			}
		}
		if _, werr := wrt.Write(buf.Bytes()); werr != nil {
			t.Errorf("failed to write JSON: %s: %w", chunk.ID, werr)
			return
		}
	}
	return outputWritersByTag, func(parentLogger logger.Logger, args base.ChunkConsumerArgs) base.ChunkConsumer {
		return newChunkSaver(args, collectChunkJSON, func() {})
	}
}

func finalizeOutputWriters(t *testing.T, outputWritersByTag map[string]*bytes.Buffer) {
	for tag, wrt := range outputWritersByTag {
		if _, err := wrt.Write([]byte("\n]\n")); err != nil {
			t.Fatalf("failed to write end bracket for tag=%s: %s", tag, err.Error())
		}
	}
}
