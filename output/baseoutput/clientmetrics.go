package baseoutput

import (
	"github.com/relex/gotils/promexporter/promext"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
)

// clientMetrics defines metrics shared by most of network-based output clients
type clientMetrics struct {
	queuedChunksLeftover    promext.RWGauge // Current numbers of chunks in the current leftovers channel
	queuedChunksPendingAck  promext.RWGauge // Current numbers of chunks waiting for ACK, including read and unread chunks by acknowledger
	networkErrorsTotal      promext.RWCounter
	nonNetworkErrorsTotal   promext.RWCounter
	openedSessionsTotal     promext.RWCounter
	forwardAttemptsTotal    promext.RWCounter
	forwardedCountTotal     promext.RWCounter
	forwardedLengthTotal    promext.RWCounter
	acknowledgedCountTotal  promext.RWCounter
	acknowledgedLengthTotal promext.RWCounter
}

func newClientMetrics(metricCreator promreg.MetricCreator) clientMetrics {
	queuedChunks := metricCreator.AddOrGetGaugeVec("output_queued_chunks", "Numbers of currently queued chunks", []string{"type"}, nil)
	return clientMetrics{
		queuedChunksLeftover:    queuedChunks.WithLabelValues("leftover"),
		queuedChunksPendingAck:  queuedChunks.WithLabelValues("pendingAck"),
		networkErrorsTotal:      metricCreator.AddOrGetCounter("output_network_errors_total", "Numbers of network errors", nil, nil),
		nonNetworkErrorsTotal:   metricCreator.AddOrGetCounter("output_nonnetwork_errors_total", "Numbers of non-network errors (auth, unexpected response, etc) from upstream", nil, nil),
		openedSessionsTotal:     metricCreator.AddOrGetCounter("output_opened_sessions_total", "Numbers of opened sessions", nil, nil),
		forwardAttemptsTotal:    metricCreator.AddOrGetCounter("output_forward_attempts_total", "Numbers of chunk forwarding attempts", nil, nil),
		forwardedCountTotal:     metricCreator.AddOrGetCounter("output_forwarded_chunks_total", "Numbers of forwarded chunks", nil, nil),
		forwardedLengthTotal:    metricCreator.AddOrGetCounter("output_forwarded_chunk_bytes_total", "Total length in bytes of forwarded chunks", nil, nil),
		acknowledgedCountTotal:  metricCreator.AddOrGetCounter("output_acknowledged_chunks_total", "Numbers of acknowledged chunks", nil, nil),
		acknowledgedLengthTotal: metricCreator.AddOrGetCounter("output_acknowledged_chunk_bytes_total", "Total length in bytes of acknowledged chunks", nil, nil),
	}
}

func (metrics *clientMetrics) OnError(err error) {
	if err != nil && util.IsNetworkError(err) {
		metrics.networkErrorsTotal.Inc()
	} else {
		metrics.nonNetworkErrorsTotal.Inc()
	}
}

func (metrics *clientMetrics) OnOpening() {
	metrics.openedSessionsTotal.Inc()
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
