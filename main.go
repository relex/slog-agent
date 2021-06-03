package main

import (
	"math/rand"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/cmd"
)

var version string

func main() {
	rand.Seed(time.Now().UnixNano()) // seed rand properly for all rand.* calls

	logger.Infof("version: %s", version)
	logger.Infof("GOMAXPROCS: %d", runtime.GOMAXPROCS(0)) // FIXME: limit GOMAXPROCS on production until go 1.16: https://github.com/golang/go/issues/28808

	registerInfoMetric()

	cmd.Execute()
}

func registerInfoMetric() {
	opts := prometheus.GaugeOpts{}
	opts.Name = "slog_agent_info"
	opts.Help = "slog-agent application information"
	gauge := prometheus.NewGaugeVec(opts, []string{"version"})
	gauge.WithLabelValues(version).Set(1)
	prometheus.MustRegister(gauge)
}
