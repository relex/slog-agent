package btest

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// A basic implementation of MultiSinkMessageReceiver for testing purpose
type logMessageAggregator struct {
	logger        logger.Logger
	outputChannel chan<- string
}

type logMessageAggregatorSink struct {
	logger        logger.Logger
	outputChannel chan<- string
}

// NewLogMessageAggregator creates a basic implementation of MultiSinkMessageReceiver, to collect incoming log messages into a single channel
func NewLogMessageAggregator(parentLogger logger.Logger) (base.MultiSinkMessageReceiver, <-chan string) {
	output := make(chan string, 100)
	return &logMessageAggregator{
		logger:        parentLogger.WithField(defs.LabelComponent, "LogMessageAggregator"),
		outputChannel: output,
	}, output
}

func (recv *logMessageAggregator) NewSink(clientAddress string, clientNumber base.ClientNumber) base.MessageReceiverSink {
	return &logMessageAggregatorSink{
		logger:        recv.logger.WithField(defs.LabelClient, clientAddress),
		outputChannel: recv.outputChannel,
	}
}

func (recv *logMessageAggregator) Destroy() {
	close(recv.outputChannel)
}

func (sess *logMessageAggregatorSink) Accept(value []byte) {
	scopy := util.DeepCopyStringFromBytes(value)
	select {
	case sess.outputChannel <- scopy:
		return
	case <-time.After(defs.IntermediateChannelTimeout):
		sess.logger.Errorf("BUG: timeout sending to channel: \"%s\". stack=%s", scopy, util.Stack())
		break
	}
}

func (sess *logMessageAggregatorSink) Flush() {
}

func (sess *logMessageAggregatorSink) Close() {
}
