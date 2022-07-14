package test

import (
	"bytes"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/output/fluentdforward"
)

type chunkSaver struct {
	args    base.ChunkConsumerArgs
	write   func(chunk base.LogChunk)
	close   func()
	stopped *channels.SignalAwaitable
}

func newChunkSaver(args base.ChunkConsumerArgs, write func(chunk base.LogChunk), close func()) base.ChunkConsumer {
	return &chunkSaver{
		args:    args,
		write:   write,
		close:   close,
		stopped: channels.NewSignalAwaitable(),
	}
}

func (saver *chunkSaver) Start() {
	go saver.run()
}

func (saver *chunkSaver) Stopped() channels.Awaitable {
	return saver.stopped
}

func (saver *chunkSaver) run() {
	defer saver.args.OnFinished()
	defer saver.stopped.Signal()
	defer saver.close()
	ch := saver.args.InputChannel
	sig := saver.args.InputClosed.Channel()
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				logger.Info("chunkSaver input closed")
				return
			}
			if chunk.Data == nil {
				logger.Panicf("received unloaded chunk id=%s", chunk.ID)
			}
			saver.write(chunk)
			saver.args.OnChunkConsumed(chunk)
		case <-sig:
			return
		}
	}
}

func openLogChunkSavers(outputPath string, outputConfig []bconfig.OutputBufferConfig) base.ChunkConsumerConstructor {
	write, close := openLogChunkConsumingFunc(outputPath)
	if write == nil {
		return nil
	}
	return func(parentLogger logger.Logger, args base.ChunkConsumerArgs) base.ChunkConsumer {
		return newChunkSaver(args, write, close)
	}
}

func openLogChunkConsumingFunc(outputPath string) (func(chunk base.LogChunk), func()) {
	if outputPath == "" {
		logger.Infof("open default output")
		return nil, nil
	}

	if outputPath == "null" {
		logger.Infof("open chunk null output")
		return func(chunk base.LogChunk) {}, func() {}
	}

	panic("unsupported output path " + outputPath)
}

func openJSONMemWriter() (func(chunk base.LogChunk), func(), func() string) {
	buffer := &bytes.Buffer{}
	buffer.WriteString("[\n")
	return func(chunk base.LogChunk) {
			_, err := fluentdforward.ConvertMsgpackToJSON(chunk, []byte(",\n"), true, buffer)
			if err != nil {
				logger.Panicf("error writing: %s", err.Error())
			}
		}, func() {
			if _, err := buffer.WriteString("\n]\n"); err != nil {
				logger.Panicf("error writing JSON end: %s", err.Error())
			}
		}, buffer.String
}
