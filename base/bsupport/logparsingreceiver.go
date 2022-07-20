package bsupport

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// LogParserConstructor represents a function to create new LogParser instances in LogParsingReceiver
type LogParserConstructor = func(parentLogger logger.Logger, inputCounter *base.LogInputCounter) base.LogParser

type logParsingReceiver struct {
	logger         logger.Logger
	createParser   LogParserConstructor
	outputReceiver base.MultiSinkBufferReceiver
	metricCreator  promreg.MetricCreator
}

type logParsingReceiverSink struct {
	logger        logger.Logger
	parser        base.LogParser
	outputSink    base.BufferReceiverSink
	bufferedLogs  []*base.LogRecord
	bufferedBytes int
	now           time.Time
	inputCounter  *base.LogInputCounter
}

// NewLogParsingReceiver creates a MultiSinkMessageReceiver to parse incoming logs, buffer them and pass to a buffer receiver
//
// Actual parsers are created on demand for each of connections
func NewLogParsingReceiver(parentLogger logger.Logger, createParser LogParserConstructor, nextReceiver base.MultiSinkBufferReceiver,
	metricCreator promreg.MetricCreator) base.MultiSinkMessageReceiver {
	return &logParsingReceiver{
		logger:         parentLogger.WithField(defs.LabelComponent, "LogParsingReceiver"),
		createParser:   createParser,
		outputReceiver: nextReceiver,
		metricCreator:  metricCreator,
	}
}

func (recv *logParsingReceiver) NewSink(clientAddress string, clientNumber base.ClientNumber) base.MessageReceiverSink {
	slogger := base.NewSinkLogger(recv.logger, clientAddress, clientNumber)
	inputCounter := base.NewLogInputCounter(recv.metricCreator)
	return &logParsingReceiverSink{
		logger:        slogger,
		parser:        recv.createParser(slogger, inputCounter),
		outputSink:    recv.outputReceiver.NewSink(clientAddress, clientNumber),
		bufferedLogs:  make([]*base.LogRecord, 0, defs.IntermediateBufferMaxNumLogs),
		bufferedBytes: 0,
		now:           time.Now(),
		inputCounter:  inputCounter,
	}
}

func (sess *logParsingReceiverSink) Accept(lines []byte) {
	record := sess.parser.Parse(lines, sess.now)
	if record == nil {
		return
	}
	sess.bufferedLogs = append(sess.bufferedLogs, record)
	sess.bufferedBytes += record.RawLength
	if sess.bufferedBytes >= defs.IntermediateBufferMaxTotalBytes ||
		len(sess.bufferedLogs) >= defs.IntermediateBufferMaxNumLogs {
		sess.sendBuffer()
	}
}

func (sess *logParsingReceiverSink) Flush() {
	sess.now = time.Now()
	if len(sess.bufferedLogs) > 0 {
		sess.sendBuffer()
	}
	sess.outputSink.Tick()
	sess.inputCounter.UpdateMetrics()
}

// Close ends this sink
func (sess *logParsingReceiverSink) Close() {
	sess.logger.Info("close")
	sess.outputSink.Close()
	sess.inputCounter.UpdateMetrics()
}

func (sess *logParsingReceiverSink) sendBuffer() {
	sess.outputSink.Accept(sess.bufferedLogs)
	sess.bufferedLogs = sess.bufferedLogs[:0]
	sess.bufferedBytes = 0
}
