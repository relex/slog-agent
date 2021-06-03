package test

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/run"
)

type logProcessor func(s []byte) *base.LogChunk
type logProcessCloser func() *base.LogChunk

func preparePipeline(configFile string, tagOverride string, metricFactory *base.MetricFactory) (bconfig.LogOutputConfig, logProcessor, logProcessCloser) {
	cfg, schema, err := run.LoadConfigFile(configFile)
	if err != nil {
		logger.Fatalf("config: %s", err.Error())
	}
	if len(cfg.Inputs) != 1 {
		logger.Fatal("only one source is supported")
	}
	inputConfig := cfg.Inputs[0].LogInputConfig
	allocator := base.NewLogAllocator(schema)
	inputCounter := base.NewLogInputCounter(metricFactory.NewSubFactory("input_", nil, nil))
	parser, perr := inputConfig.NewParser(logger.Root(), allocator, schema, inputCounter)
	if perr != nil {
		panic(perr)
	}
	procCounter := base.NewLogProcessCounter(metricFactory.NewSubFactory("process_", nil, nil), schema, schema.MustCreateFieldLocators(cfg.MetricKeys))
	transforms := bsupport.NewTransformsFromConfig(cfg.Transformations, schema, logger.Root(), procCounter)
	serializer := cfg.Output.NewSerializer(logger.Root(), schema, allocator)
	chunkMaker := cfg.Output.NewChunkMaker(logger.Root(), tagOverride)
	now := time.Now()
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
	return cfg.Output, process, endProcess
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
