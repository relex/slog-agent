package baseoutput

import (
	"github.com/relex/gotils/promexporter"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
)

// clientMetrics defines metrics shared by most of network-based output clients
type clientMetrics struct {
	queuedChunksLeftover    promexporter.RWGauge // Current numbers of chunks in the current leftovers channel
	queuedChunksPendingAck  promexporter.RWGauge // Current numbers of chunks waiting for ACK, including read and unread chunks by acknowledger
	networkErrorsTotal      promexporter.RWCounter
	nonNetworkErrorsTotal   promexporter.RWCounter
	forwardAttemptsTotal    promexporter.RWCounter
	forwardedCountTotal     promexporter.RWCounter
	forwardedLengthTotal    promexporter.RWCounter
	acknowledgedCountTotal  promexporter.RWCounter
	acknowledgedLengthTotal promexporter.RWCounter
}

func newClientMetrics(metricFactory *base.MetricFactory) clientMetrics {
	queuedChunks := metricFactory.AddOrGetGaugeVec("output_queued_chunks", "Numbers of currently queued chunks", []string{"type"}, nil)
	return clientMetrics{
		queuedChunksLeftover:    queuedChunks.WithLabelValues("leftover"),
		queuedChunksPendingAck:  queuedChunks.WithLabelValues("pendingAck"),
		networkErrorsTotal:      metricFactory.AddOrGetCounter("output_network_errors_total", "Numbers of network errors", nil, nil),
		nonNetworkErrorsTotal:   metricFactory.AddOrGetCounter("output_nonnetwork_errors_total", "Numbers of non-network errors (auth, unexpected response, etc) from upstream", nil, nil),
		forwardAttemptsTotal:    metricFactory.AddOrGetCounter("output_forward_attempts_total", "Numbers of chunk forwarding attempts", nil, nil),
		forwardedCountTotal:     metricFactory.AddOrGetCounter("output_forwarded_chunks_total", "Numbers of forwarded chunks", nil, nil),
		forwardedLengthTotal:    metricFactory.AddOrGetCounter("output_forwarded_chunk_bytes_total", "Total length in bytes of forwarded chunks", nil, nil),
		acknowledgedCountTotal:  metricFactory.AddOrGetCounter("output_acknowledged_chunks_total", "Numbers of acknowledged chunks", nil, nil),
		acknowledgedLengthTotal: metricFactory.AddOrGetCounter("output_acknowledged_chunk_bytes_total", "Total length in bytes of acknowledged chunks", nil, nil),
	}
}

func (metrics *clientMetrics) OnError(err error) {
	if err != nil && util.IsNetworkError(err) {
		metrics.networkErrorsTotal.Inc()
	} else {
		metrics.nonNetworkErrorsTotal.Inc()
	}
}

func (metrics *clientMetrics) OnForwarding(chunk base.LogChunk) {
	metrics.forwardAttemptsTotal.Inc()
}

func (metrics *clientMetrics) OnForwarded(chunk base.LogChunk) {
	metrics.forwardedCountTotal.Inc()
	metrics.forwardedLengthTotal.Add(uint64(len(chunk.Data)))
	metrics.queuedChunksPendingAck.Inc()
}

func (metrics *clientMetrics) OnAcknowledged(chunk base.LogChunk) {
	metrics.acknowledgedCountTotal.Inc()
	metrics.acknowledgedLengthTotal.Add(uint64(len(chunk.Data)))
	metrics.queuedChunksPendingAck.Dec()
}

func (metrics *clientMetrics) OnLeftoverPopped(chunk base.LogChunk) {
	metrics.queuedChunksLeftover.Dec()
}

func (metrics *clientMetrics) OnSessionEnded(previousLeftovers int, unacked int, newLeftovers int) {
	metrics.queuedChunksPendingAck.Sub(int64(unacked))
	metrics.queuedChunksLeftover.Add(int64(newLeftovers - previousLeftovers))
}
