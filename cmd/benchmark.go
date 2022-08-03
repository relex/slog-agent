package cmd

import (
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/test"
)

type benchmarkCommandState struct {
	Input  string `help:"Input file path or wildcard pattern (RFC 5424 Syslog)."`
	Output string `help:"Output file path:\n'': (empty) forward as configured\n'null': abandon all output\nmerged JSON file, e.g. /tmp/all-logs.json\nper-chunk JSON files, e.g. /tmp/chunk-%s.json\nper-chunk msgpack files, e.g. /tmp/chunk-%s.bin"`
	Repeat int    `help:"Repeat times"`
	Config string `help:"Configuration file path"`
}

var benchCmd = benchmarkCommandState{
	Input:  "testdata/development/*.log",
	Output: "null",
	Config: "testdata/config_sample.yml",
	Repeat: 100,
}

func (cmd *benchmarkCommandState) runBenchmarkPipelineCommand(_ []string) {
	defs.EnableTestMode()
	test.RunBenchmarkPipeline(cmd.Input, cmd.Output, cmd.Repeat, cmd.Config)
}

func (cmd *benchmarkCommandState) runBenchmarkAgentCommand(_ []string) {
	defs.EnableTestMode()
	test.RunBenchmarkAgent(cmd.Input, cmd.Output, cmd.Repeat, cmd.Config)
}
