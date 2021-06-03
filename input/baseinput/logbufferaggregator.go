package baseinput

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

type logBufferAggregator struct {
	logger        logger.Logger
	outputChannel chan<- []*base.LogRecord
}

type logBufferAggregatorChannel struct {
	logger        logger.Logger
	outputChannel chan<- []*base.LogRecord
	sendTimeout   *time.Timer
}

// NewLogBufferAggregator creates a MultiChannelReceiver to collect incoming logs to a single channel for test purpose
func NewLogBufferAggregator(parentLogger logger.Logger) (base.MultiChannelBufferReceiver, <-chan []*base.LogRecord) {
	ch := make(chan []*base.LogRecord, defs.IntermediateBufferedChannelSize)
	return &logBufferAggregator{
		logger:        parentLogger.WithField(defs.LabelComponent, "LogBufferAggregator"),
		outputChannel: ch,
	}, ch
}

func (recv *logBufferAggregator) NewChannel(id string) base.BufferReceiverChannel {
	slogger := recv.logger.WithFields(logger.Fields{
		defs.LabelPart:   "channel",
		defs.LabelRemote: id,
	})
	return &logBufferAggregatorChannel{
		logger:        slogger,
		outputChannel: recv.outputChannel,
		sendTimeout:   time.NewTimer(defs.IntermediateChannelTimeout),
	}
}

func (recv *logBufferAggregator) Destroy() {
	close(recv.outputChannel)
	recv.logger.Infof("destroy channel, remaining=%d", len(recv.outputChannel))
}

func (sess *logBufferAggregatorChannel) Accept(buffer []*base.LogRecord) {
	if len(buffer) == 0 {
		return
	}
	reusableBuffer := bsupport.CopyLogBuffer(buffer)
	select {
	case sess.outputChannel <- reusableBuffer:
		break
	case <-sess.sendTimeout.C:
		sess.logger.Errorf("BUG: timeout flushing: %d records", len(reusableBuffer))
		break
	}
}

func (sess *logBufferAggregatorChannel) Tick() {
	util.ResetTimer(sess.sendTimeout, defs.IntermediateChannelTimeout)
}

// Close ends this channel
func (sess *logBufferAggregatorChannel) Close() {
	sess.logger.Info("close")
}
