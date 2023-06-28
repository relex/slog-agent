package test

import (
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/run"
	"github.com/samber/lo"
)

type testPipeline struct {
	deallocator       base.LogAllocator
	fallbackTimestamp time.Time
	inputCounter      *base.LogInputCounterSet
	inputParser       base.LogParser
	outputNames       []string
	outputDecoders    []base.ChunkDecoder
	outputSavers      []chunkSaver
	procCounter       *base.LogProcessCounterSet
	transforms        []base.LogTransformFunc
	serializers       []base.LogSerializer
	chunkMakers       []base.LogChunkMaker
}

func preparePipeline(configFile string, tagOverride string, metricCreator promreg.MetricCreator,
	newChunkSaver func(outputName string) chunkSaver,
) *testPipeline {
	conf, schema, stats, err := run.ParseConfigFile(configFile)
	if err != nil {
		logger.Fatalf("config: %s", err.Error())
	}
	stats.Log(logger.Root())
	if len(conf.Inputs) != 1 {
		logger.Fatal("only one input source is supported")
	}

	inputConfig := conf.Inputs[0].Value // we support only one input for testing
	allocator := base.NewLogAllocator(schema, len(conf.OutputBuffersPairs))
	inputCounter := base.NewLogInputCounter(metricCreator.AddOrGetPrefix("input_", nil, nil))

	parser, perr := inputConfig.NewParser(logger.Root(), allocator, schema, inputCounter)
	if perr != nil {
		logger.Panic("failed to create parser: ", perr)
	}

	outputNames := lo.Map(conf.OutputBuffersPairs, func(pair bconfig.OutputBufferConfig, _ int) string {
		return pair.Name
	})
	procCounter := base.NewLogProcessCounter(
		metricCreator.AddOrGetPrefix("process_", nil, nil),
		schema,
		schema.MustCreateFieldLocators(conf.MetricKeys),
		outputNames,
	)

	return &testPipeline{
		deallocator:       *allocator,
		fallbackTimestamp: time.Now(),
		inputCounter:      inputCounter,
		inputParser:       parser,
		outputNames:       outputNames,
		outputDecoders: lo.Map(conf.OutputBuffersPairs, func(pair bconfig.OutputBufferConfig, _ int) base.ChunkDecoder {
			return pair.OutputConfig.Value
		}),
		outputSavers: lo.Map(outputNames, func(outputName string, _ int) chunkSaver {
			return newChunkSaver(outputName)
		}),
		procCounter: procCounter,
		transforms:  bsupport.NewTransformsFromConfig(conf.Transformations, schema, logger.Root(), procCounter),
		serializers: lo.Map(conf.OutputBuffersPairs, func(pair bconfig.OutputBufferConfig, _ int) base.LogSerializer {
			return pair.OutputConfig.Value.NewSerializer(logger.Root(), schema, tagOverride)
		}),
		chunkMakers: lo.Map(conf.OutputBuffersPairs, func(pair bconfig.OutputBufferConfig, _ int) base.LogChunkMaker {
			return pair.OutputConfig.Value.NewChunkMaker(logger.Root(), tagOverride)
		}),
	}
}

func (p *testPipeline) GetOutputNames() []string {
	return p.outputNames
}

func (p *testPipeline) Run(inputLines [][]byte, repeat int) {
	for _, saver := range p.outputSavers {
		defer saver.Close()
	}

	chunkHolder := make([]*base.LogChunk, len(p.chunkMakers))
	for n := 0; n < repeat; n++ {
		for _, line := range inputLines {
			p.process(line, chunkHolder)
			p.save(chunkHolder, p.outputSavers)
		}
	}
	p.flush(chunkHolder)
	p.save(chunkHolder, p.outputSavers)
}

func (p *testPipeline) process(s []byte, chunkHolder []*base.LogChunk) {
	if s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	record := p.inputParser.Parse(s, p.fallbackTimestamp)
	if record == nil {
		return
	}
	icounter := p.procCounter.SelectMetricKeySet(record)
	if bsupport.RunTransforms(record, p.transforms) == base.DROP {
		icounter.CountRecordDrop(record)
		p.deallocator.Release(record)
		return
	}
	icounter.CountRecordPass(record)
	for outputIndex, serializer := range p.serializers {
		stream := serializer.SerializeRecord(record)
		p.procCounter.CountStream(outputIndex, stream)

		maybeChunk := p.chunkMakers[outputIndex].WriteStream(stream)
		if maybeChunk != nil {
			p.procCounter.CountChunk(outputIndex, maybeChunk)
		}
		chunkHolder[outputIndex] = maybeChunk
	}
}

func (p *testPipeline) flush(chunkHolder []*base.LogChunk) {
	for outputIndex, chunkMaker := range p.chunkMakers {
		maybeChunk := chunkMaker.FlushBuffer()
		if maybeChunk != nil {
			p.procCounter.CountChunk(outputIndex, maybeChunk)
		}
		chunkHolder[outputIndex] = maybeChunk
	}
	p.inputCounter.UpdateMetrics()
	p.procCounter.UpdateMetrics()
}

func (p *testPipeline) save(chunkHolder []*base.LogChunk, outputSavers []chunkSaver) {
	for outputIndex, decoder := range p.outputDecoders {
		maybeChunk := chunkHolder[outputIndex]
		if maybeChunk != nil {
			outputSavers[outputIndex].Write(*maybeChunk, decoder)
		}
	}
}
