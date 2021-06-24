package bsupport

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

// LogParserConstructor represents a function to create new LogParser instances in LogParsingReceiver
type LogParserConstructor = func(parentLogger logger.Logger, inputCounter *base.LogInputCounter) base.LogParser

type logParsingReceiver struct {
	logger         logger.Logger
	createParser   LogParserConstructor
	outputReceiver base.MultiChannelBufferReceiver
	metricFactory  *base.MetricFactory
}

type logParsingReceiverChannel struct {
	logger        logger.Logger
	parser        base.LogParser
	outputChannel base.BufferReceiverChannel
	bufferedLogs  []*base.LogRecord
	bufferedBytes int
	now           time.Time
	inputCounter  *base.LogInputCounter
}

// NewLogParsingReceiver creates a MultiChannelStringReceiver to parse incoming logs, buffer them and pass to a buffer receiver
//
// Actual parsers are created on demand for each of connections
func NewLogParsingReceiver(parentLogger logger.Logger, createParser LogParserConstructor, nextReceiver base.MultiChannelBufferReceiver,
	metricFactory *base.MetricFactory) base.MultiChannelStringReceiver {
	return &logParsingReceiver{
		logger:         parentLogger.WithField(defs.LabelComponent, "LogParsingReceiver"),
		createParser:   createParser,
		outputReceiver: nextReceiver,
		metricFactory:  metricFactory,
	}
}

func (recv *logParsingReceiver) NewChannel(id string) base.StringReceiverChannel {
	slogger := recv.logger.WithFields(logger.Fields{
		defs.LabelPart:   "channel",
		defs.LabelRemote: id,
	})
	inputCounter := base.NewLogInputCounter(recv.metricFactory)
	return &logParsingReceiverChannel{
		logger:        slogger,
		parser:        recv.createParser(slogger, inputCounter),
		outputChannel: recv.outputReceiver.NewChannel(id),
		bufferedLogs:  make([]*base.LogRecord, 0, defs.IntermediateBufferMaxNumLogs),
		bufferedBytes: 0,
		now:           time.Now(),
		inputCounter:  inputCounter,
	}
}

func (sess *logParsingReceiverChannel) Accept(lines []byte) {
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

func (sess *logParsingReceiverChannel) Flush() {
	sess.now = time.Now()
	if len(sess.bufferedLogs) > 0 {
		sess.sendBuffer()
	}
	sess.outputChannel.Tick()
	sess.inputCounter.UpdateMetrics()
}

// Close ends this channel
func (sess *logParsingReceiverChannel) Close() {
	sess.logger.Info("close")
	sess.outputChannel.Close()
	sess.inputCounter.UpdateMetrics()
}

func (sess *logParsingReceiverChannel) sendBuffer() {
	sess.outputChannel.Accept(sess.bufferedLogs)
	sess.bufferedLogs = sess.bufferedLogs[:0]
	sess.bufferedBytes = 0
}
