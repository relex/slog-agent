// Package hybridbuffer provides an ChunkBufferer implementation which keeps N numbers of unsent chunks in memory and
// starts saving to and loading from the queue directory when the limit is reached.
//
// At shutdown, all unsent chunks are saved for recovery next time
//
// The bufferer creates subdirs for each buffer ID under root path. A subdir name is made of sanitized buffer/pipeline
// ID and hash to prevent collision, while the original ID is stored as an extended attribute on the subdir itself.
//
// If for any reason the queue directory cannot be created or accessed, the buffer drops all chunks after the limit is
// reached and logs errors; at shutdown all pending chunks would be held until they're sent or timed out.
package hybridbuffer

import "github.com/relex/gotils/promexporter/promreg"

func makeBufferMetricCreator(parentMetricCreator promreg.MetricCreator) promreg.MetricCreator {
	return parentMetricCreator.AddOrGetPrefix("", []string{"storage"}, []string{"hybridBuffer"})
}
