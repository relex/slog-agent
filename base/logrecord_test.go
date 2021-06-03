package base

import (
	"strconv"
	"testing"
)

const testLogMessage = "Lorem ipsum dolor sit amet, consectetur adipiscing elit"
const testLogSeverity = "unknown"

func BenchmarkLogRecordOpByMap(b *testing.B) {
	for iter := 0; iter < b.N; iter++ {
		log := make(map[string]string, 20)
		log["pri"] = strconv.FormatInt(int64(iter), 10)
		log["message"] = testLogMessage
		log["severity"] = testLogSeverity
		log["log"] = log["message"]
		log["level"] = log["severity"]
		delete(log, "message")
		delete(log, "severity")
	}
}

func BenchmarkLogRecordOpBySliceIndexed(b *testing.B) {
	fPri := 0
	fMessage := 1
	fSeverity := 2
	fLog := 3
	fLevel := 4
	for iter := 0; iter < b.N; iter++ {
		log := make([]string, 20)
		log[fPri] = strconv.FormatInt(int64(iter), 10)
		log[fMessage] = testLogMessage
		log[fSeverity] = testLogSeverity
		log[fLog] = log[fMessage]
		log[fLevel] = log[fSeverity]
		log[fMessage] = ""
		log[fSeverity] = ""
	}
}

func BenchmarkLogRecordOpBySliceIndexedRef(b *testing.B) {
	fPri := &fieldLocator{"pri", 0}
	fMessage := &fieldLocator{"msg", 1}
	fSeverity := &fieldLocator{"severity", 2}
	fLog := &fieldLocator{"log", 3}
	fLevel := &fieldLocator{"level", 4}
	for iter := 0; iter < b.N; iter++ {
		log := make([]string, 20)
		log[fPri.index] = strconv.FormatInt(int64(iter), 10)
		log[fMessage.index] = testLogMessage
		log[fSeverity.index] = testLogSeverity
		log[fLog.index] = log[fMessage.index]
		log[fLevel.index] = log[fSeverity.index]
		log[fMessage.index] = ""
		log[fSeverity.index] = ""
	}
}

func BenchmarkLogRecordOpBySliceWrapped(b *testing.B) {
	fPri := fieldLocator{"pri", 0}
	fMessage := fieldLocator{"msg", 1}
	fSeverity := fieldLocator{"severity", 2}
	fLog := fieldLocator{"log", 3}
	fLevel := fieldLocator{"level", 4}
	for iter := 0; iter < b.N; iter++ {
		log := make([]string, 20)
		fPri.set(log, strconv.FormatInt(int64(iter), 10))
		fMessage.set(log, testLogMessage)
		fSeverity.set(log, testLogSeverity)
		fLog.set(log, fMessage.get(log))
		fLevel.set(log, fSeverity.get(log))
		fMessage.del(log)
		fSeverity.del(log)
	}
}

func BenchmarkLogRecordOpBySliceWrappedRef(b *testing.B) {
	ls := fieldList{
		fPri:      fieldLocator{"pri", 0},
		fMessage:  fieldLocator{"msg", 1},
		fSeverity: fieldLocator{"severity", 2},
		fLog:      fieldLocator{"log", 3},
		fLevel:    fieldLocator{"level", 4},
	}
	for iter := 0; iter < b.N; iter++ {
		log := make([]string, 20)
		ls.fPri.set(log, strconv.FormatInt(int64(iter), 10))
		ls.fMessage.set(log, testLogMessage)
		ls.fSeverity.set(log, testLogSeverity)
		ls.fLog.set(log, ls.fMessage.get(log))
		ls.fLevel.set(log, ls.fSeverity.get(log))
		ls.fMessage.del(log)
		ls.fSeverity.del(log)
	}
}

type fieldList struct {
	fPri      fieldLocator
	fMessage  fieldLocator
	fSeverity fieldLocator
	fLog      fieldLocator
	fLevel    fieldLocator
}

type fieldLocator struct {
	name  string
	index int
}

func (loc fieldLocator) get(fields []string) string {
	return fields[loc.index]
}

func (loc fieldLocator) set(fields []string, val string) {
	fields[loc.index] = val
}

func (loc fieldLocator) del(fields []string) {
	fields[loc.index] = ""
}
