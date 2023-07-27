package test

import (
	"os"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/testdata"
	"github.com/relex/slog-agent/util"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestDataGeneration(t *testing.T) {
	if !util.IsTestGenerationMode() {
		return
	}
	t.Log("regenerate log outputs...")
	mfactory := promreg.NewMetricFactory("testpipeline_", nil, nil)
	for _, inPath := range testdata.ListInputFiles(t) {
		t.Log("regenerate log output for " + inPath)
		inLines := loadInputRecords(inPath)
		inTitle := testdata.GetInputTitle(t, inPath)
		pipeline := preparePipeline(testdata.GetConfigPath(), inTitle, mfactory, func(outName string) chunkSaver {
			return newChunkSaver(outName, testdata.GetOutputFilenamePattern(t, inPath))
		})
		pipeline.Run(inLines, 1)
	}

	assert.NoError(t, os.WriteFile("../testdata/development/all-pipeline.prom", []byte(promext.DumpMetrics("", true, true, mfactory)), 0644))
}

func TestPipeline(t *testing.T) {
	if util.IsTestGenerationMode() {
		return
	}
	mfactory := promreg.NewMetricFactory("testpipeline_", nil, nil)
	for _, inPath := range testdata.ListInputFiles(t) {
		localInPath := inPath
		inTitle := testdata.GetInputTitle(t, inPath)
		t.Run(inTitle, func(tt *testing.T) {
			// output name => expected contents
			expectedOutputs := lo.MapValues(testdata.ListOutputNamesAndFiles(tt, inTitle), func(expectedOutPath, _ string) string {
				expectedContent, eOutErr := os.ReadFile(expectedOutPath)
				assert.NoError(t, eOutErr, "reading "+expectedOutPath)
				return string(expectedContent)
			})

			// output name => actual content getter
			actualOutputGetters := make(map[string]func() string)
			pipeline := preparePipeline(testdata.GetConfigPath(), inTitle, mfactory, func(outName string) chunkSaver {
				saver, getter := newInMemoryChunkSaver(logger.WithField("output", outName))
				actualOutputGetters[outName] = getter
				return saver
			})
			pipeline.Run(loadInputRecords(localInPath), 1)

			for outName, expected := range expectedOutputs {
				outGetter := actualOutputGetters[outName]
				if assert.NotNil(t, outGetter, "known output %s", outName) {
					assert.Equal(t, expected, outGetter(), "known output %s", outName)
				}
			}

			for outName := range actualOutputGetters {
				_, ok := expectedOutputs[outName]
				assert.True(t, ok, "unexpected output %s", outName)
			}
		})
	}
	t.Run("check metrics", func(tt *testing.T) {
		expectedMetrics, err := os.ReadFile("../testdata/development/all-pipeline.prom")
		if assert.NoError(tt, err) {
			assert.Equal(tt, string(expectedMetrics), promext.DumpMetrics("", true, true, mfactory))
		}
	})
}
