package textractspecial

import (
	"regexp"
	"testing"
)

const testMessageWithComponent = `[BackupManager] - Unable to open /var/backup/appServ/bar [OPEN]: permission denied`
const testMessageWithNamedField = `Unable to open, name=var_backup_appServ_bar`
const testSourceWithTimestamp = `folderA/cron.log.202011011234`
const testSourceWithoutTimestamp = `folderB/main.log`

func BenchmarkExtractingComponentByRegexp(b *testing.B) {
	re := regexp.MustCompile(`^(\[(?P<component>[^ \]]*) *\] - )?(?P<log>.*)$`)
	indexComp := re.SubexpIndex("component")
	indexLog := re.SubexpIndex("log")
	b.ResetTimer()
	lastComp := ""
	lastLog := ""
	for i := 0; i < b.N; i++ {
		m := re.FindStringSubmatch(testMessageWithComponent)
		lastComp = m[indexComp]
		lastLog = m[indexLog]
	}
	b.Logf("last '%s', '%s'", lastComp, lastLog)
}

func BenchmarkExtractingComponentByFSearch(b *testing.B) {
	prefix := "["
	postfix := "] - "
	b.ResetTimer()
	lastComp := ""
	lastLog := ""
	for i := 0; i < b.N; i++ {
		comp, log := extractLabelAtStart(testMessageWithComponent, prefix, postfix, 50, nil)
		lastComp = comp
		lastLog = log
	}
	b.Logf("last '%s', '%s'", lastComp, lastLog)
}

func BenchmarkExtractingNamedFieldByRegexp(b *testing.B) {
	re := regexp.MustCompile(`^(?P<log>.*)(, (?P<key>[a-zA-Z0-9_]+)=(?P<val>[a-zA-Z0-9_]+))?$`)
	indexLog := re.SubexpIndex("log")
	indexKey := re.SubexpIndex("key")
	indexVal := re.SubexpIndex("val")
	b.ResetTimer()
	lastLog := ""
	lastKey := ""
	lastVal := ""
	for i := 0; i < b.N; i++ {
		m := re.FindStringSubmatch(testMessageWithNamedField)
		lastLog = m[indexLog]
		lastKey = m[indexKey]
		lastVal = m[indexVal]
	}
	b.Logf("last '%s'='%s', '%s'", lastKey, lastVal, lastLog)
}

func BenchmarkExtractingNamedFieldByCustom(b *testing.B) {
	// ["a-z", "A-Z", "0-9", "_=*"]
	tbl := make([]bool, 256)
	for c := 'a'; c <= 'z'; c++ {
		tbl[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		tbl[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		tbl[c] = true
	}
	tbl['_'] = true
	tbl['='] = true
	b.ResetTimer()
	lastLog := ""
	lastKey := ""
	lastVal := ""
	for i := 0; i < b.N; i++ {
		keyPair, log := extractLabelAtEnd(testMessageWithNamedField, ", ", "", 50, tbl)
		key, val := extractLabelAtStart(keyPair, "", "=", 50, nil)
		lastLog = log
		lastKey = key
		lastVal = val
	}
	b.Logf("last '%s'='%s', '%s'", lastKey, lastVal, lastLog)
}

func BenchmarkParsingSourceByRegexp(b *testing.B) {
	re := regexp.MustCompile(`^((?P<folder>[^ /]+)/)?(?P<source>[^ ]*?)(\.(?P<timestamp>-?[0-9]+))?$`)
	indexFolder := re.SubexpIndex("folder")
	indexSource := re.SubexpIndex("source")
	indexTimestamp := re.SubexpIndex("timestamp")
	m := map[string]string{
		"BenchmarkParsingSourceByRegexp with timestamp id":    testSourceWithTimestamp,
		"BenchmarkParsingSourceByRegexp without timestamp id": testSourceWithoutTimestamp,
	}
	for name, text := range m {
		localText := text
		b.Run(name, func(bb *testing.B) {
			lastFolder := ""
			lastSource := ""
			lastTimestamp := ""
			for i := 0; i < bb.N; i++ {
				m := re.FindStringSubmatch(localText)
				lastFolder = m[indexFolder]
				lastSource = m[indexSource]
				lastTimestamp = m[indexTimestamp]
			}
			bb.Logf("last folder=%s, source=%s, timestamp=%s", lastFolder, lastSource, lastTimestamp)
		})
	}
}

func BenchmarkParsingSourceByCustom(b *testing.B) {
	var numberTable = make([]bool, 256)
	for c := '0'; c <= '9'; c++ {
		numberTable[c] = true
	}
	m := map[string]string{
		"BenchmarkParsingSourceByFSearch with timestamp":    testSourceWithTimestamp,
		"BenchmarkParsingSourceByFSearch without timestamp": testSourceWithoutTimestamp,
	}
	for name, text := range m {
		localText := text
		b.Run(name, func(bb *testing.B) {
			lastFolder := ""
			lastSource := ""
			lastTimestamp := ""
			for i := 0; i < bb.N; i++ {
				folder, sourceWithTimestamp := extractLabelAtStart(localText, "", "/", 100, nil)
				timestamp, source := extractLabelAtEnd(sourceWithTimestamp, ".", "", 50, numberTable)
				lastFolder = folder
				lastSource = source
				lastTimestamp = timestamp
			}
			bb.Logf("last folder=%s, source=%s, timestamp=%s", lastFolder, lastSource, lastTimestamp)
		})
	}
}
