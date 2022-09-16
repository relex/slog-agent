package test

import (
	"bytes"
	"os"
	"strings"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/util"
)

// loadInputRecords loads sample syslog file into an array of records (NOT lines) for pipeline test
func loadInputRecords(inputPath string) [][]byte {
	inputData, _ := loadInput(inputPath)
	inputRemaining := inputData
	inputLines := make([][]byte, 0, len(inputData)/100)
	sep := []byte("\n<")
	eol := []byte("\n")
	for len(inputRemaining) > 0 {
		next := bytes.Index(inputRemaining, sep)
		if next >= 0 {
			ln := inputRemaining[:next]
			inputLines = append(inputLines, ln)
			inputRemaining = inputRemaining[next+1:]
		} else {
			ln := bytes.TrimSuffix(inputRemaining, eol)
			inputLines = append(inputLines, ln)
			break
		}
	}
	return inputLines
}

// loadInput loads sample syslog file into raw data block that can be fed to syslog input, and also count the records (NOT lines)
func loadInput(inputPath string) ([]byte, int) {
	pathList, gerr := util.ListFiles(inputPath)
	if gerr != nil {
		logger.Fatal(gerr)
	} else if len(pathList) == 0 {
		logger.Fatal("no input files")
	}
	data := make([]byte, 0)
	numLines := 0
	for _, path := range pathList {
		content, err := os.ReadFile(path)
		if err != nil {
			logger.Fatalf("error reading %s: %v", path, err)
		}
		if len(content) == 0 {
			continue
		}
		nl := strings.Count(string(content), "\n<") + 1 // first line counts as a record whether it has headers or not
		data = append(data, content...)
		numLines += nl
		logger.Infof("loaded %s: %d records, %d bytes", path, nl, len(content))
	}
	return data, numLines
}
