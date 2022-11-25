package test

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/testdata"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestAgent(t *testing.T) {
	regenOutput := util.IsTestGenerationMode()

	// launch agent
	loader, confErr := run.NewLoaderFromConfigFile(testdata.GetConfigPath(), "testagent_")
	assert.Nil(t, confErr)
	loader.ConfigStats.Log(logger.WithField("test", t.Name()))

	collectedMultiOutputs, outputOverrideCreator := prepareInMemoryConsumerOverride(t)
	// Override tag for output splitting and keys (labelsets) for distribution: order of logs would be messed up if keys are different
	agt := startAgent(loader, outputOverrideCreator, []string{"host"}, "$host")

	// feed input
	inputDataByTag, expectedMultiOutputResults := getInputDataAndExpectedOutputsByTag(t)
	inputWorkerCounter := &sync.WaitGroup{}
	for tag, input := range inputDataByTag {
		logger.Infof("launching writer for input tag=%s len=%d", tag, len(input))
		inputWorkerCounter.Add(1)
		go func(input []byte) {
			feedInputToBenchmarkAgent(agt.Address(), input, 1)
			inputWorkerCounter.Done()
		}(input)
	}

	// wait for ending
	logger.Infof("waiting for %d input writers...", len(inputDataByTag))
	inputWorkerCounter.Wait()
	time.Sleep(1 * time.Second) // TODO: find out why socket accept, receive and allow conn close before go code is invoked?
	logger.Info("stopping agent...")
	agt.StopAndWait()
	// finalizeOutputBuffers(t, outputBuffersByTag)

	// compare outputs
	if !regenOutput {
		t.Run("check known outputs", func(tt *testing.T) {
			for outName, expectedByTag := range expectedMultiOutputResults {
				actualByTag, actualOutOk := collectedMultiOutputs[outName]
				assert.True(tt, actualOutOk, outName)

				for tag, expected := range expectedByTag {
					buf, ok := actualByTag[tag]
					if assert.True(tt, ok, tag, outName+"-"+tag) {
						assert.Equal(tt, expected, buf.String(), outName+"-"+tag)
					}
				}
			}
		})
		t.Run("check unexpected outputs", func(tt *testing.T) {
			for outName, actualByTag := range collectedMultiOutputs {
				expectedByTag, expectedOutOk := expectedMultiOutputResults[outName]
				assert.True(tt, expectedOutOk, outName)

				for tag := range actualByTag {
					_, ok := expectedByTag[tag]
					assert.True(tt, ok, tag)
				}
			}
		})
		t.Run("check metrics", func(tt *testing.T) {
			expectedMetrics, err := os.ReadFile("../testdata/development/all-agent.prom")
			if assert.Nil(tt, err) {
				assert.Equal(tt, string(expectedMetrics), promext.DumpMetricsFrom("", true, true, agt.GetMetricQuerier()))
			}
		})
	} else {
		// output JSONs are to be generated from pipeline test, not here
		assert.Nil(t, os.WriteFile("../testdata/development/all-agent.prom", []byte(promext.DumpMetricsFrom("", true, true, agt.GetMetricQuerier())), 0644))
	}
}

type outputStringByTag map[string]string
type expectedMultiOutput map[string]outputStringByTag // output => tag => JSON string

func getInputDataAndExpectedOutputsByTag(t *testing.T) (map[string][]byte, expectedMultiOutput) {
	inputFiles := testdata.ListInputFiles(t)
	assert.NotZero(t, len(inputFiles))

	inputDataByTag := make(map[string][]byte)
	expectedMultiOutputResults := make(expectedMultiOutput) // output name => tag => expected
	for _, inPath := range inputFiles {
		title := testdata.GetInputTitle(t, inPath)
		{
			inputData, ierr := os.ReadFile(inPath)
			assert.Nil(t, ierr)
			inputDataByTag[title] = inputData
		}
		{
			outputToPaths := testdata.ListOutputNamesAndFiles(t, title)
			for outputName, outputPath := range outputToPaths {
				expectedByTag, ok := expectedMultiOutputResults[outputName]
				if !ok {
					expectedByTag = make(outputStringByTag)
					expectedMultiOutputResults[outputName] = expectedByTag
				}
				expectedResult, oerr := os.ReadFile(outputPath)
				assert.Nil(t, oerr)
				expectedByTag[title] = string(expectedResult)
			}
		}
	}
	return inputDataByTag, expectedMultiOutputResults
}
