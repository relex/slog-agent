package util

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/defs"
)

func init() {
	_ = pprof.Handler // to trigger registrations under "/debug/pprof/"
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/api/v1/metrics/prometheus", promhttp.Handler()) // for fluent-bit compatibility
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
<html>
	<head>
		<title>slog-agent metrics listener</title>
	</head>
	<body>
		<h1>Metrics listener for slog-agent</h1>
		<ul>
			<li><a href='/debug/pprof'>/debug/pprof</a></li>
			<li><a href='/metrics'>/metrics</a></li>
		</ul>
	</body>
</html>`)
	})
}

// LaunchMetricsListener starts a HTTP server for Prometheus metrics
func LaunchMetricsListener(address string) *http.Server {
	mlogger := logger.WithField(defs.LabelComponent, "MetricsListener")
	server := &http.Server{}
	server.Addr = address
	go func() {
		mlogger.Infof("listening on %s for metrics...", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			mlogger.Error("Prometheus listener error: ", err)
		}
	}()
	return server
}

// SumMetricValues sums all the values of a given Prometheus Collector (GaugeVec or CounterVec)
func SumMetricValues(c prometheus.Collector) float64 {
	// modified from github.com/prometheus/client_golang/prometheus/testutil.ToFloat64
	var (
		mList = make([]prometheus.Metric, 0, 100)
		mChan = make(chan prometheus.Metric)
		done  = make(chan struct{})
	)
	go func() {
		for m := range mChan {
			mList = append(mList, m)
		}
		close(done)
	}()
	c.Collect(mChan)
	close(mChan)
	<-done

	sum := 0.0
	for _, m := range mList {
		pb := &dto.Metric{}
		if err := m.Write(pb); err != nil {
			logger.Errorf("failed to read metric '%s': %s", m.Desc(), err.Error())
		}
		if pb.Gauge != nil {
			sum += pb.Gauge.GetValue()
		}
		if pb.Counter != nil {
			sum += pb.Counter.GetValue()
		}
		if pb.Untyped != nil {
			sum += pb.Untyped.GetValue()
		}

	}
	return sum
}
