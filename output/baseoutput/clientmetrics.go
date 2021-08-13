package baseoutput

import (
	"github.com/relex/gotils/promexporter"
	"github.com/relex/slog-agent/base"
)

// ClientMetrics defines metrics shared by most of network-based output clients
type ClientMetrics struct {
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

func NewClientMetrics(metricFactory *base.MetricFactory) ClientMetrics {
	queuedChunks := metricFactory.AddOrGetGaugeVec("output_queued_chunks", "Numbers of currently queued chunks", []string{"type"}, nil)
	return ClientMetrics{
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

func (metrics *ClientMetrics) IncrementNetworkErrors() {
	metrics.networkErrorsTotal.Inc()
}

func (metrics *ClientMetrics) IncrementNonNetworkErrors() {
	metrics.nonNetworkErrorsTotal.Inc()
}

func (metrics *ClientMetrics) OnForwarding(chunk base.LogChunk) {
	metrics.forwardAttemptsTotal.Inc()
}

func (metrics *ClientMetrics) OnForwarded(chunk base.LogChunk) {
	metrics.forwardedCountTotal.Inc()
	metrics.forwardedLengthTotal.Add(uint64(len(chunk.Data)))
	metrics.queuedChunksPendingAck.Inc()
}

func (metrics *ClientMetrics) OnAcknowledged(chunk base.LogChunk) {
	metrics.acknowledgedCountTotal.Inc()
	metrics.acknowledgedLengthTotal.Add(uint64(len(chunk.Data)))
	metrics.queuedChunksPendingAck.Dec()
}

func (metrics *ClientMetrics) OnLeftoverPopped(chunk base.LogChunk) {
	metrics.queuedChunksLeftover.Dec()
}

func (metrics *ClientMetrics) OnSessionEnded(previousLeftovers int, unacked int, newLeftovers int) {
	metrics.queuedChunksPendingAck.Sub(int64(unacked))
	metrics.queuedChunksLeftover.Add(int64(newLeftovers - previousLeftovers))
}
