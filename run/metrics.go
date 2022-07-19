package run

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	reloadSuccessCounter prometheus.Counter
	reloadFailureCounter prometheus.Counter
)

func init() {
	opts := prometheus.CounterOpts{}
	opts.Name = "slogagent_reloads_total"
	opts.Help = "Numbers of reloads"
	vec := prometheus.NewCounterVec(opts, []string{"status"})
	prometheus.MustRegister(vec)

	reloadSuccessCounter = vec.WithLabelValues("success")
	reloadFailureCounter = vec.WithLabelValues("failure")
}
