# Manual integration testing

A basic test setup for slog-agent is like:

1. Start upstream: run `fluentlibtool server` provided by [fluentlib](https://github.com/relex/fluentlib) as a fake Fluentd
2. Start agent: run `BUILD/slog-agent run --config=testdata/config_sample.yml`
3. Feed input: `cat testdata/development/*.log > /dev/tcp/127.0.0.1/5140`

## Current coverage

- Buffer (HybridBuffer): recovery mechanism
- Input (Syslog): parsing, error handling
- Orchestrate (ByKeySetOrchestrator): distribution by key-set (= label sets)
- Rewrite and Transform: fully covered by unit tests
- Run: config initialization and reloading

## Areas lacking tests

### Orchestrate: parallel sub-pipelines

It's a replacement / augment of ByKeySetOrchestrator to split each key-set pipeline into fixed numbers of parallel
subpipelines, which receive log chunks from the ParallelDistributor which splits traffic.

Due to the requirements from Loki storage, log chunks have to be collected in the original order though they may be
processed in parallel. Mutexes in *OrderedLogBuffer* are used to synchronize and ensure the order.

The code involves:

  - base/bsupport/orderedlogprocessingworker.go: OrderedLogProcessingWorker
  - base/bsupport/pipelineparallel.go: ParallelPipeline*
  - orchestrate/obase/distributor.go: ParallelDistributor

### Output: error recovery

The normal code-path for output is already covered, but not any of error related paths that are critical to production
environments. Those code are concentrated in:

  - `output/baseoutput/clientsession.go`
  - `output/baseoutput/clientworker.go`

Some of the expected errors and special code-paths are:

  - Fluentd keeps receiving chunks but not responding
  - Log chunks sent unsuccessfully
  - Log chunks sent but not acknowledged
  - Interruption in recovery stage before all leftover chunks from the previous session can be sent
  - Sessions reach their max duration and abort for reconnection
  - The need to abort connection when a shutdown signal is received, in order to interrupt any ongoing read/write

Except the session max-duration which can be tested by using very short `upstream/maxDuration` in config, the rest may
be helped by emulation of random network or Fluentd errors in [fluentlib](https://github.com/relex/fluentlib) and
`--test_mode` of the agent, but there are no automated tests as it'd require not only to verify the final output but
also certain code paths being touched and the contents of certain variables at certain stages.
