package test

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/run"
)

type logProcessor func(s []byte) *base.LogChunk
type logProcessCloser func() *base.LogChunk

func preparePipeline(configFile string, tagOverride string, metricCreator promreg.MetricCreator) (bconfig.LogOutputConfig, logProcessor, logProcessCloser) {
	conf, schema, stats, err := run.ParseConfigFile(configFile)
	if err != nil {
		logger.Fatalf("config: %s", err.Error())
	}
	stats.Log(logger.Root())
	if len(conf.Inputs) != 1 {
		logger.Fatal("only one input source is supported")
	}

	inputConfig := conf.Inputs[0].Value // we support only one input for testing
	allocator := base.NewLogAllocator(schema)
	inputCounter := base.NewLogInputCounter(metricCreator.AddOrGetPrefix("input_", nil, nil))

	parser, perr := inputConfig.NewParser(logger.Root(), allocator, schema, inputCounter)
	if perr != nil {
		logger.Panic("failed to create parser: ", perr)
	}

	procCounter := base.NewLogProcessCounter(metricCreator.AddOrGetPrefix("process_", nil, nil), schema, schema.MustCreateFieldLocators(conf.MetricKeys))
	transforms := bsupport.NewTransformsFromConfig(conf.Transformations, schema, logger.Root(), procCounter)
	serializer := conf.Output.Value.NewSerializer(logger.Root(), schema, allocator)
	chunkMaker := conf.Output.Value.NewChunkMaker(logger.Root(), tagOverride)

	now := time.Now() // fallback timestamp
	process := func(s []byte) *base.LogChunk {
		if s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}
		record := parser.Parse(s, now)
		if record == nil {
			return nil
		}
		icounter := procCounter.SelectInputCounter(record)
		if bsupport.RunTransforms(record, transforms) == base.DROP {
			icounter.CountRecordDrop(record)
			allocator.Release(record)
			return nil
		}
		icounter.CountRecordPass(record)
		stream := serializer.SerializeRecord(record)
		procCounter.CountStream(stream)
		maybeChunk := chunkMaker.WriteStream(stream)
		if maybeChunk != nil {
			procCounter.CountChunk(maybeChunk)
		}
		return maybeChunk
	}

	endProcess := func() *base.LogChunk {
		maybeChunk := chunkMaker.FlushBuffer()
		if maybeChunk != nil {
			procCounter.CountChunk(maybeChunk)
		}
		inputCounter.UpdateMetrics()
		procCounter.UpdateMetrics()
		return maybeChunk
	}

	return conf.Output.Value, process, endProcess
}

func runPipeline(process logProcessor, endProcess logProcessCloser, inputLines [][]byte, repeat int, writeChunk func(chunk base.LogChunk)) {
	for n := 0; n < repeat; n++ {
		for _, line := range inputLines {
			maybeChunk := process(line)
			if maybeChunk != nil {
				writeChunk(*maybeChunk)
			}
		}
	}
	maybeLastChunk := endProcess()
	if maybeLastChunk != nil {
		writeChunk(*maybeLastChunk)
	}
}
