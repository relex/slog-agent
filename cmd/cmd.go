// Package cmd provides list of commands including self-benchmarks and tools
package cmd

import (
	"github.com/relex/gotils/config"
)

func init() {
	config.AddParentCmdWithArgs("", "slog-agent collects syslog logs, transform and forward them to log-pipeline", &rootCmd, rootCmd.preRun, rootCmd.postRun)
	config.AddCmdWithArgs("benchmark <type> ...", "Run benchmark of specified type", &benchCmd, nil)
	config.AddCmdWithArgs("benchmark pipeline ...", "Benchmark pipeline with null or file output", nil, benchCmd.runBenchmarkPipelineCommand)
	config.AddCmdWithArgs("benchmark agent ...", "Benchmark agent with null or file output", nil, benchCmd.runBenchmarkAgentCommand)
	config.AddCmdWithArgs("run ...", "Run agent", &runCmd, runCmd.run)
}

// Execute parses the command line and runs the specified command
func Execute() {
	// trigger init

	config.Execute()
}
