package baseinput

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// A basic implementation of MultiChannelReceiver to collect all inputs for testing purpose
type logStringAggregator struct {
	logger        logger.Logger
	outputChannel chan<- string
}

type logStringAggregatorChannel struct {
	logger        logger.Logger
	outputChannel chan<- string
	sendTimeout   *time.Timer
}

// NewLogStringAggregator creates a basic implementation of MultiChannelReceiver, to collect all input data into a single channel
func NewLogStringAggregator(parentLogger logger.Logger) (base.MultiChannelStringReceiver, <-chan string) {
	output := make(chan string, 100)
	return &logStringAggregator{
		logger:        parentLogger.WithField(defs.LabelComponent, "LogStringAggregator"),
		outputChannel: output,
	}, output
}

func (recv *logStringAggregator) NewChannel(id string) base.StringReceiverChannel {
	return &logStringAggregatorChannel{
		logger:        recv.logger.WithField(defs.LabelRemote, id),
		outputChannel: recv.outputChannel,
		sendTimeout:   time.NewTimer(defs.IntermediateChannelTimeout),
	}
}

func (recv *logStringAggregator) Destroy() {
	close(recv.outputChannel)
}

func (sess *logStringAggregatorChannel) Accept(value []byte) {
	scopy := util.DeepCopyStringFromBytes(value)
	select {
	case sess.outputChannel <- scopy:
		return
	case <-sess.sendTimeout.C:
		sess.logger.Error("BUG: timeout sending to channel: ", scopy)
		break
	}
}

func (sess *logStringAggregatorChannel) Flush() {
	util.ResetTimer(sess.sendTimeout, defs.IntermediateChannelTimeout)
}

func (sess *logStringAggregatorChannel) Close() {
}
