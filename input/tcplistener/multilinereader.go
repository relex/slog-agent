package tcplistener

import (
	"bytes"

	"github.com/relex/slog-agent/util"
)

type ioReader func(p []byte) (n int, err error)
type recordConsumer func(s []byte)
type headTester func(s []byte) bool

// multiLineReader keeps entire multi-line records on buffer for zero heap alloc and minimal moving
// The reader is designed for malformed syslog, for example:
//
//   <163>1 2019-08-15T15:50:46.866915+03:00 local my-app 123 fn - First line
//   Second line
//   Third line
//   <162>1 2019-08-15T15:51:46.866915+03:00 local my-app 123 fn - Next message
//
// We can only identify the starting line of a multi-line record, not the end.
// The end is done by periodic timneout/flush, assuming all lines in a multi-line log would be sent in a very short time
// and any long pause would indicate the start of next record.
//
// Multi-line reading is optional and not needed for our own logs
type multiLineReader struct {
	readInput       ioReader       // io.Reader.Read
	testRecordStart headTester     // test whether a line is the start of a valid record, not including newline
	consumeRecord   recordConsumer // callback to consume a record, not including last newline and may be oversized
	softRecordLimit int            // soft limit of record length, may not be applied to every consumeRecord calls
	buffer          []byte         // preallocated buffer
	offsetSearch    int            // point to start of the last line, could be end of buffer
	offsetAppend    int            // point to end of buffer
}

func newMultiLineReader(read ioReader, test headTester, minBufferSize, softRecordLimit int, consume recordConsumer) *multiLineReader {
	return &multiLineReader{
		readInput:       read,
		testRecordStart: test,
		consumeRecord:   consume,
		softRecordLimit: softRecordLimit,
		buffer:          make([]byte, util.MaxInt(minBufferSize, softRecordLimit*3)),
		offsetSearch:    0,
		offsetAppend:    0,
	}
}

// Read reads next block to buffer and consumes any valid records in buffer
// It always reads as much as the buffer allows
func (mlr *multiLineReader) Read() error {
	n, err := mlr.readInput(mlr.buffer[mlr.offsetAppend:])
	if n > 0 {
		bufferedLength := n + mlr.offsetAppend
		mlr.processBuffer(bufferedLength)
	}
	return err
}

// Flush considers buffered multi-line record completed and consumes it if valid
func (mlr *multiLineReader) Flush() {
	buffer := mlr.buffer[:mlr.offsetAppend]
	n := bytes.LastIndexByte(buffer, '\n')
	if n == -1 {
		return
	}
	record := buffer[:n]
	if len(record) > 0 && mlr.testRecordStart(record) {
		mlr.consumeRecord(record)
	}
	// relocate unfinished record to the beginning
	mlr.offsetAppend = copy(mlr.buffer, buffer[n+1:])
	mlr.offsetSearch = 0
}

// FlushAll is like Flush but including the last unfinished line, to be done before shutdown
func (mlr *multiLineReader) FlushAll() {
	record := mlr.buffer[:mlr.offsetAppend]
	if len(record) > 0 {
		// cut trailing newline
		if record[len(record)-1] == '\n' {
			record = record[:len(record)-1]
		}
		if mlr.testRecordStart(record) {
			mlr.consumeRecord(record)
		}
	}
	mlr.offsetAppend = 0
	mlr.offsetSearch = 0
}

func (mlr *multiLineReader) processBuffer(bufferEnd int) {
	recordStart := 0
	searchStart := mlr.offsetSearch
	buffer := mlr.buffer[:bufferEnd]
	test := mlr.testRecordStart
	for {
		nextEndRel := bytes.IndexByte(buffer[searchStart:], '\n')
		if nextEndRel == -1 {
			break
		}
		nextEnd := nextEndRel + searchStart
		// only test if there are previous lines, laid out as: [prev record L1, '\n', prev record L2, '\n', next record L1, '\n']
		if searchStart > 0 && searchStart < nextEnd {
			nextLine := buffer[searchStart:nextEnd]
			if test(nextLine) {
				prevRecord := buffer[recordStart : searchStart-1]
				mlr.consumeRecord(prevRecord)
				recordStart = searchStart
			}
		}
		searchStart = nextEnd + 1
	}
	if recordStart > 0 {
		// relocate unfinished record to the beginning
		mlr.offsetAppend = copy(mlr.buffer, buffer[recordStart:])
		mlr.offsetSearch = searchStart - recordStart
	} else {
		mlr.offsetAppend = bufferEnd
		mlr.offsetSearch = searchStart
	}
	mlr.checkOverflow()
}

func (mlr *multiLineReader) checkOverflow() {
	// if we have room for another record of max length, just leave it
	if len(mlr.buffer)-mlr.offsetAppend >= mlr.softRecordLimit {
		return
	}
	buffer := mlr.buffer[:mlr.offsetAppend]
	if searchStart := mlr.offsetSearch; searchStart > 0 {
		if nextRecord := buffer[searchStart:]; mlr.testRecordStart(nextRecord) {
			if prevRecord := buffer[:searchStart-1]; mlr.testRecordStart(prevRecord) {
				mlr.consumeRecord(prevRecord)
			}
			mlr.consumeRecord(nextRecord)
			goto RESET
		}
	}
	if wholeRecord := buffer; mlr.testRecordStart(wholeRecord) {
		mlr.consumeRecord(wholeRecord)
	}
RESET:
	mlr.offsetAppend = 0
	mlr.offsetSearch = 0
}
