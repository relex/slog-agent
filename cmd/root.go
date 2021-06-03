package cmd

import (
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"github.com/relex/gotils/logger"
)

type rootCommandState struct {
	CPUProfile string `name:"cpuprofile" help:"Write CPU profile to file."`
	MemProfile string `name:"memprofile" help:"Write memory profile to file."`
	Trace      string `help:"Write trace to file."`

	cpuProfileFile *os.File
	memProfileFile *os.File
	traceFile      *os.File
}

var rootCmd rootCommandState

func (cmd *rootCommandState) preRun() {
	if cmd.CPUProfile != "" {
		f, err := os.Create(cmd.CPUProfile)
		if err != nil {
			logger.Fatalf("failed to create CPU profile %s: %s", cmd.CPUProfile, err.Error())
		}

		logger.Infof("start CPU profiling %s", cmd.CPUProfile)
		if err := pprof.StartCPUProfile(f); err != nil {
			logger.Fatalf("failed to start CPU profiling: %s", err.Error())
		}

		cmd.cpuProfileFile = f
	}

	if cmd.MemProfile != "" {
		f, err := os.Create(cmd.MemProfile)
		if err != nil {
			logger.Fatalf("failed to create memory profile %s: %s", cmd.MemProfile, err.Error())
		}

		logger.Infof("start memory profiling %s", cmd.MemProfile)

		cmd.memProfileFile = f
	}

	if cmd.Trace != "" {
		f, err := os.Create(cmd.Trace)
		if err != nil {
			logger.Fatalf("failed to create trace %s: %s", cmd.Trace, err.Error())
		}

		logger.Infof("start tracing %s", cmd.Trace)
		if err := trace.Start(f); err != nil {
			logger.Fatalf("failed to start tracing: %s", err.Error())
		}

		cmd.traceFile = f
	}
}

func (cmd *rootCommandState) postRun() {
	if cmd.cpuProfileFile != nil {
		pprof.StopCPUProfile()
		cmd.cpuProfileFile.Close()
	}

	if cmd.memProfileFile != nil {
		runtime.GC()
		if err := pprof.WriteHeapProfile(cmd.memProfileFile); err != nil {
			logger.Errorf("failed to write memory profile: %s", err.Error())
		}
		cmd.memProfileFile.Close()
	}

	if cmd.traceFile != nil {
		trace.Stop()
		cmd.traceFile.Close()
	}
}
