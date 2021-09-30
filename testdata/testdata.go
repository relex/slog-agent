// Package testdata provides access to shared sample logs and config for testing
package testdata

import (
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

var inputExtPattern = regexp.MustCompile(`-input\.log$`)

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

func ListInputFiles(t *testing.T, pattern string) []string {
	fullPattern := filepath.Join(absoluteDirPath, "development", pattern+"-input.log")

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

func GetOutputFilename(t *testing.T, fn string) string {
	outFn := inputExtPattern.ReplaceAllString(fn, "-output.json")
	if outFn == fn {
		t.Fatalf("invalid input filename %s", fn)
	}
	return outFn
}
