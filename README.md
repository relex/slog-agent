# slog-agent

A log agent designed to process and filter massive amounts of logs in reat-time and forward to upsteam (fluentd)


## What we built this for

We have hundreds of thousands of application logs *per second* that need to be processed or filtered as quickly as
possible, *for each server*.

At the target rate of one million logs per second, every steps could be bottlenecks and conventional log processors
are not designed to handle that sort of traffic. This agent is built to be extremely efficient, both memory and CPU
wise, and also to be able to scale up to multiple CPU cores *efficiently*, at the cost of everything else.

A possibly baised and unfair comparison of this vs Lua transform with fluent-bit, is roughly 0.5M log/s from network
input, processed and gzipped at 1:20-50 ratio (2 cores), vs 50K log/s from file and uncompressed (one core) for the
same processing steps. We also tested [Vector](https://vector.dev/) with similar but worse results.


## What you need to adopt this

You need basic understanding of [Go](https://golang.org/), to be ready to write new transforms and dig into profiling
reports.

Things are slow on generic log processors for very good reasons - For example, a simple matching by regular expression
could be 50 times slower than a [special glob pattern](https://github.com/gobwas/glob), and allocates tons of buffers
in memory heap which then need more CPU time to be GC'ed. The boundary crossing scripting interface is another
bottleneck, with marshalling and unmarshalling of each records that could cost more than the script execution itself.

Without any of such generic and flexible transforms and parsers, everything needs to be done in manually written code,
or blocks of code that can be assembled together - which is essentially what this log agent provides, a base and blocks
of code for you to build high performance log processors - but only if you need that kind of performance. The design is
pluggable and the program is largely configurable, but you're going to run into situations which can only be solved by
writing new code.


## Features

- Input: RFC 5424 Syslog protocol via TCP, with experimental multiline support
- Transforms: field extraction and creations, drop, truncate, if/switch, email redaction
- Buffering: hybrid disk+memory buffering - compressed and only persisted when necessary
- Output: Fluentd Forward protocol, both compressed and uncompressed. Single output only.
- Metrics: Prometheus metrics to count logs and log size by key fields (e.g. vhost + log level + filename)

Dynamic fields are not supported - All fields must be known in configuration because they're packed in arrays that can
be accessed without hashmap lookup.

"tags" or similar concept doesn't exist here. Instead there are "if" and "switch-case" matching field values.

See the [sample configurations](testdata/config_sample.yml) for full features.

#### Performance and Backpressure

Logs are compressed and saved in chunk files if output cannot clear the logs fast enough. The maximum numbers of
pending chunks for each pipeline (key field set) are limited and defined in [defs/params.go](defs/params.go).

Input would be paused if logs cannot be processed fast enough - since RFC 5424 doesn't support any pause mechanism,
it'd likely cause internal errors on both the agent and the logging application, but would not affect other
applications' logging if pipelines are properly set-up / isolated (e.g. by app-name and vhost).

For a typical server CPU (e.g. Xeon, 2GHz), a single pipeline / core should be able to handle at least:

- 300-500K log/s for small logs, around 100-200 bytes each including syslog headers
- 200K log/s or 400MB/s for larger logs 

Note on servers with more than a few dozens of CPU cores, an optimal `GOMAXPROCS` has to be measured and set for
production workload, until https://github.com/golang/go/issues/28808 is resolved

## Build

Requires [gotils](https://github.com/relex/gotils) which provides build tools

```bash
make
make test
```
## Operation manual

#### Configuration

See [sample configurations](testdata/config_sample.yml).

Experimental configuration reloading is supported by starting with `--allow_reload` and sending `SIGHUP`; See
[testdata/config_sample.yml] for details on which sections may be reconfigured. In general everything after inputs
are re-configurable. If reconfiguration fails, errors are logged and the agent would continue to run with old
configuration, without any side-effect.

Note after successful reloading, some of previous logs may be sent to upstream again if they hadn't been acknowledged
in time.

The metric family `slogagent_reloads_total` counts sucesses and failures of reconfigurations.

Currently it is not possible to recover previously queued logs if `orchestration/keys` have been changed.

#### Runtime diagnosis

- `SIGHUP` aborts and recreates all pipelines with new config loaded from the same file. Incoming connections are unaffected.
- `SIGUSR1` recreates all outgoing connections or sessions gracefully.
- http://localhost:METRICS_PORT/ provides Golang's builtin debug functions in addition to metrics, such as stackdump and profiling.

## Development

#### Mark inlinable code

Add `xx:inline` comment on the same line as function declaration

```go
func (s *xLogSchema) GetFieldName(index int) string { // xx:inline
```

If this function is too complex to be inlined, build would fail with a warning.

#### Re-generate templated source (.tpl.go)

```bash
make gen
```

#### Re-generate expected output in integration tests

```bash
make test-gen
```

## Runtime Diagnosis

Prometheus listener address (default *:9335*) exposes go's `debug/pprof` in addition to metrics, which can dump
goroutine stacks.

Options:

- `--cpuprofile FILE_PATH`: enable GO CPU profiling, with some overhead
- `--memprofile FILE_PATH`: enable GO CPU profiling
- `--trace FILE_PATH`: enable GO tracing

## Benchmark & Profiling

Example:

```bash
LOG_LEVEL=warn time BUILD/slog-agent benchmark agent --input 'testdata/development/*.log' --repeat 250000 --config testdata/config_sample.yml --output null --cpuprofile /tmp/agent.cpu --memprofile /tmp/agent.mem
go tool pprof -http=:8080 BUILD/slog-agent /tmp/agent.cpu
```

`--output` supports several formats:

- `` (empty): default to forward to fluentd as defined in config.
  Chunks may not be fully sent when shutdown and unsent chunks would be saved for next run.
- `null`: no output. Results are compressed as in normal routine, counted and then dropped.
- `.../%s`: create fluentd forward message files each of individual chunks at the path (`%s` as chunk ID). The dir must exist first.
- `.../%s.json`: create JSON files each of individual chunks at the path (`%s` as chunk ID). The dir must exist first.

fluentd forward message files can be examined by [fluentlibtool](https://github.com/relex/fluentlib)

## Internals

See [DESIGN](DESIGN.md)

#### Key dependencies

- [fluentlib](https://github.com/relex/fluentlib) for fluentd forward protocol, fake server and dump tool for testing.
- [klauspost' compress library](github.com/klauspost/compress) for fast gzip compression which is absolutely critical
  to the agent: always benchmark before upgrade. *compression takes 1/2 to 1/3 of CPU time in our environments*
- [YAML v3](gopkg.in/yaml.v3) required for custom tags in configuration. `KnownFields` is still not working and it
  cannot check non-existent or misspelled YAML properties.

#### Go upgrades

Some of code is based on the internal behaviors of go runtime and marked as `GO_INTERNAL` in comments:

- util/syncwaitgroup.go `PeekWaitGroup()`: peek the count number from `sync.WaitGroup`
- util/strings.go `StringFromBytes()`: make string from mutable bytes without copying, like `strings.Builder` but
  accepts mutable arrays (*dangerous*)

They need to be re-checked for any major upgrade of go for compatibility

## Authors

Special thanks to _Henrik Sjöström_ for his guiding on Go optimization, integration testing and invaluable suggestions
on performant design.
