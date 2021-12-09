package btest

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// A basic implementation of MultiSinkBufferReceiver for testing purpose
type logBufferAggregator struct {
	logger        logger.Logger
	outputChannel chan<- []*base.LogRecord
}

type logBufferAggregatorSink struct {
	logger        logger.Logger
	outputChannel chan<- []*base.LogRecord
	sendTimeout   *time.Timer
}

// NewLogBufferAggregator creates a basic implementation of MultiSinkBufferReceiver, to collect incoming log batches into a single channel
func NewLogBufferAggregator(parentLogger logger.Logger) (base.MultiSinkBufferReceiver, <-chan []*base.LogRecord) {
	ch := make(chan []*base.LogRecord, defs.IntermediateBufferedChannelSize)
	return &logBufferAggregator{
		logger:        parentLogger.WithField(defs.LabelComponent, "LogBufferAggregator"),
		outputChannel: ch,
	}, ch
}

func (recv *logBufferAggregator) NewSink(clientAddress string, clientNumber base.ClientNumber) base.BufferReceiverSink {
	slogger := base.NewSinkLogger(recv.logger, clientAddress, clientNumber)
	return &logBufferAggregatorSink{
		logger:        slogger,
		outputChannel: recv.outputChannel,
		sendTimeout:   time.NewTimer(defs.IntermediateChannelTimeout),
	}
}

func (recv *logBufferAggregator) Destroy() {
	close(recv.outputChannel)
	recv.logger.Infof("destroy channel, remaining=%d", len(recv.outputChannel))
}

func (sess *logBufferAggregatorSink) Accept(buffer []*base.LogRecord) {
	if len(buffer) == 0 {
		return
	}
	reusableBuffer := bsupport.CopyLogBuffer(buffer)
	select {
	case sess.outputChannel <- reusableBuffer:
		break
	case <-sess.sendTimeout.C:
		sess.logger.Errorf("BUG: timeout flushing: %d records. stack=%s", len(reusableBuffer), util.Stack())
		break
	}
}

func (sess *logBufferAggregatorSink) Tick() {
	util.ResetTimer(sess.sendTimeout, defs.IntermediateChannelTimeout)
}

// Close ends this channel
func (sess *logBufferAggregatorSink) Close() {
	sess.logger.Info("close")
}
