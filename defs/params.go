package defs

import (
	"time"
)

var (
	// InputLogMaxMessageBytes defines the maximum length of a log message, if such a field exists (parser dependent)
	// If the limit is exceeded, the message should be truncated and may be recorded in metrics
	InputLogMaxMessageBytes = 1 * 1024 * 1024

	// InputLogMinMessageBytesToPool defines the minimum length of a log message to start using object pooling
	// Only pool large buffers since sync.Pool takes time
	InputLogMinMessageBytesToPool = 1024

	// InputFlushInterval defines how long to call flush from input if no log is received
	// It's used to trigger flushing in all receivers, e.g. multiLineReader waiting for next line in a multi-line syslog message
	// The value affects the delay of logs, as they may not be processed until flush is called
	InputFlushInterval = 500 * time.Millisecond

	// ListenerLineBufferSize defines the buffer size in bytes to read one syslog line
	// If the size is insufficient to hold incoming line, listener switches to a dynamic buffer.
	ListenerLineBufferSize = InputLogMaxMessageBytes * 4

	// IntermediateBufferMaxNumLogs defines the maximum numbers of log records to buffer at input before flushing through go channels
	// The value affects size of buffers passing down channels
	// Larger number puts pressure on GC and makes context switching worse,
	// e.g. 50000 causes +2.5 sec than 1000 for 10M small logs or 2.3GB, while transform takes less 7 sec
	// Related to GO scheduler and internals, re-evaluate the value for go upgrade
	IntermediateBufferMaxNumLogs = 500

	// IntermediateBufferMaxTotalBytes defines the how many bytes can be allowed in the buffer of input log records before forced flushing
	IntermediateBufferMaxTotalBytes = 4 * 1024 * 1024

	// IntermediateBufferedChannelSize defines the size of internal buffered channels meant to contain temporary data
	// 0 = unbuffered channels
	// Higher value improves parallelization when ByKeySetParallelOrchestrator is used
	IntermediateBufferedChannelSize = 1

	// IntermediateChannelTimeout defines the timeout of intermediate channel reads and writes.
	// There is no recovery without data loss and it should be treated as a bug if such timeout happens at runtime
	IntermediateChannelTimeout = 20 * time.Second

	// IntermediateFlushInterval defines how often for intermediate workers to flush their own states
	// For example, to flush buffer streams into output chunks, or to update internal timer
	IntermediateFlushInterval = 1 * time.Second

	// ParallelizationBufferMaxNumLogs defines the numbers of logs to process before switching to next parallel worker
	// Lower value improves parallelization but reduces the size of output chunk files
	// e.g. 25000 logs * 250 bytes avg / 20 compression ratio ~= 200-300KB chunk(s)
	ParallelizationBufferMaxNumLogs = 25000

	// ParallelizationBufferMaxTotalBytes defines the numbers of raw input bytes to process before switching to next parallel worker
	// The value can be slightly higher than OutputChunkMaxDataBytes
	ParallelizationBufferMaxTotalBytes = OutputChunkMaxDataBytes + OutputChunkMaxDataBytes/4

	// OutputChunkMaxDataBytes defines the max uncompressed data size of a LogChunk, not including necessary headers
	// The value must be lower than the maximum buffer length acceptable by upstream
	OutputChunkMaxDataBytes = 8 * 1024 * 1024

	// BufferMaxNumChunksInQueue is the max numbers of of loaded and unloaded chunks to be held in a queue,
	// equal to the max numbers of queued files on disk, because at least all the filepaths need to be held in channel
	// New logs are dropped when the limit is reached
	// e.g. 200000 * 300K chunk files ~= 56GB compressed or 1.1TB uncompressed
	BufferMaxNumChunksInQueue = 200000

	// BufferMaxNumChunksInMemory is the max numbers of of loaded chunks to be held in a queue
	// Disk persistance starts when output queue length hits value / 2, because length reading is delayed / inaccurate
	BufferMaxNumChunksInMemory = 500

	// BufferShutDownTimeout is the duration to wait for LogBuffer to save or send all pending log chunks when shutdown
	BufferShutDownTimeout = ForwarderBatchAckTimeout + IntermediateChannelTimeout*2
)

var (
	// ForwarderMaxBuffersForAck is the max number of chunk files waiting in cleanup stage before ACK
	ForwarderMaxBuffersForAck = 10

	// ForwarderConnectionTimeout is for establishing a TCP connection to upstream
	ForwarderConnectionTimeout = 60 * time.Second

	// ForwarderHandshakeTimeout is for TLS handshake with upstream
	ForwarderHandshakeTimeout = ForwarderConnectionTimeout + ForwarderConnectionTimeout/2

	// ForwarderBatchSendMinimumSpeed is the minimum speed in bytes/sec to calculate timeout
	// Actual timeout for sending is [base] + [packet length] / [minimal speed per]
	ForwarderBatchSendMinimumSpeed = 10 * 1024

	// ForwarderBatchSendTimeoutBase is how long to wait at least for sending one batch.
	ForwarderBatchSendTimeoutBase = ForwarderConnectionTimeout + ForwarderConnectionTimeout/2

	// ForwarderBatchAckTimeout is how long to wait for receiving one batch ACK.
	// The value depends on how fast upstream can process all logs before buffering, related to OutputChunkMaxDataBytes
	ForwarderBatchAckTimeout = ForwarderConnectionTimeout + 60*time.Second

	// ForwarderAckerStopTimeout defines how long to wait for acknowledger to stop.
	// The timeout isn't supposed to be reached but a precaution in case some bug hangs the forwarder client
	// Need to wait until the current ACK to finish or timeout in order to collect leftovers properly
	ForwarderAckerStopTimeout = ForwarderBatchAckTimeout + IntermediateChannelTimeout

	// ForwarderRetryInterval is how long to wait after a connection is interrupted.
	ForwarderRetryInterval = 10 * time.Second

	// ForwarderPingInterval is how often to send an empty request for keep-alive / ping purpose
	ForwarderPingInterval = 20 * time.Second
)

// For tests
const (
	TestReadTimeout = 5 * time.Second
)

// EnableTestMode turns on test mode with very short timeout and minimal retry delay
func EnableTestMode() {
	ForwarderConnectionTimeout = 1 * time.Second
	ForwarderHandshakeTimeout = 2 * time.Second
	ForwarderBatchSendTimeoutBase = 3 * time.Second
	ForwarderBatchAckTimeout = 3 * time.Second
	ForwarderRetryInterval = 100 * time.Millisecond
	ForwarderPingInterval = 100 * time.Millisecond
}
