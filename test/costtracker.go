package test

import (
	"runtime"
	"syscall"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/util"
)

// CostTracker tracks CPU usage and memory allocations
type CostTracker struct {
	initRealTime      time.Time
	initUserTime      time.Time
	initSystemTime    time.Time
	initNumHeapAllocs uint64
}

// CostReport contains measurements since .Start()
type CostReport struct {
	RealTime      time.Duration
	UserTime      time.Duration
	SystemTime    time.Duration
	NumHeapAllocs uint64
	GCCPUFraction float64
}

// NewCostTracker creates a cost tracker and starts tracking
func NewCostTracker() *CostTracker {
	runtime.GC()
	ct := &CostTracker{}
	ct.initRealTime = time.Now()
	{
		var rusage syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
			logger.Panic("failed to get resource usage: ", err)
		}
		ct.initUserTime = util.TimeFromTimeval(rusage.Utime)
		ct.initSystemTime = util.TimeFromTimeval(rusage.Stime)
	}
	{
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		ct.initNumHeapAllocs = memStats.Mallocs
	}
	return ct
}

// Report reports measurements since .Start() was called
func (ct *CostTracker) Report() CostReport {
	runtime.GC()
	var report CostReport
	{
		report.RealTime = time.Since(ct.initRealTime)
	}
	{
		var rusage syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
			logger.Panic("failed to get resource usage: ", err)
		}
		report.UserTime = util.TimeFromTimeval(rusage.Utime).Sub(ct.initUserTime)
		report.SystemTime = util.TimeFromTimeval(rusage.Stime).Sub(ct.initSystemTime)
	}
	{
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		report.NumHeapAllocs = memStats.Mallocs - ct.initNumHeapAllocs
		report.GCCPUFraction = memStats.GCCPUFraction
	}
	return report
}
