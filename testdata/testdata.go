// Package testdata provides access to shared sample logs and config for testing
package testdata

import (
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/samber/lo"
)

var absoluteDirPath string

func init() {
	_, thisFile, _, _ := runtime.Caller(0)
	absoluteDirPath = filepath.Dir(thisFile)
}

func GetConfigPath() string {
	return filepath.Join(absoluteDirPath, "config_sample.yml")
}

func GetConfigDumpPath() string {
	return filepath.Join(absoluteDirPath, "config_sample_dump.yml")
}

var inputExtPattern = regexp.MustCompile(`-input\.log$`)

func ListInputFiles(t *testing.T) []string {
	fullPattern := filepath.Join(absoluteDirPath, "development", "*-input.log")

	inFiles, globErr := filepath.Glob(fullPattern)
	if globErr != nil {
		t.Fatalf("failed to scan test files at path %s: %v", fullPattern, globErr)
	}
	if len(inFiles) == 0 {
		t.Fatalf("failed to find test files at path %s: no match", fullPattern)
	}
	return inFiles
}

func GetInputTitle(t *testing.T, fn string) string {
	title := inputExtPattern.ReplaceAllString(fn, "")
	if title == fn {
		t.Fatalf("invalid input filename %s", fn)
	}
	return filepath.Base(title)
}

func GetOutputFilenamePattern(t *testing.T, fn string) string {
	outFn := inputExtPattern.ReplaceAllString(fn, "-output-$$OUTPUT.json")
	if outFn == fn {
		t.Fatalf("invalid input filename %s", fn)
	}
	return outFn
}

var outputPathPattern = regexp.MustCompile(`^.*/[^/]+-output-(.+)\.json$`)

func ListOutputNamesAndFiles(t *testing.T, inputTitle string) map[string]string {
	fullPattern := filepath.Join(absoluteDirPath, "development", inputTitle+"-output-*.json")

	outFiles, globErr := filepath.Glob(fullPattern)
	if globErr != nil {
		t.Fatalf("failed to scan expected output files at path %s: %v", fullPattern, globErr)
	}
	if len(outFiles) == 0 {
		t.Fatalf("failed to find expected output files at path %s: no match", fullPattern)
	}

	return lo.KeyBy(outFiles, func(path string) string {
		return outputPathPattern.ReplaceAllString(path, "$1")
	})
}
