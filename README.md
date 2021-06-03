# slog-agent

A log agent designed to process and filter massive amounts of logs in reat-time and forward to fluentd


## What's we built this for

We have hundreds of thousands of application logs *per second* that need to be processed or dropped as quickly as
possible, *for each server*.

At the target rate of one million logs per second, every steps are bottlenecks and conventional log processors are
simply not designed to handle that sort of load. This agent is built to be as efficient as possible, both memory
and CPU wise, and to be able to take advantage of hundreds of CPU cores *efficiently*, at the cost of everything else.

PS: A possibly baised and unfair comparison of this vs Lua transform with fluent-bit, is roughly 0.5M log/s with gzip
compression (2 cores), vs 50K log/s uncompressed (one core) for the same processing we used. We also tested
[Vector](https://vector.dev/) with similar but worse results.


## What you need to use this

You need basic understanding of [Go](https://golang.org/), be ready to write a new transform and dig into profiling
reports.

Things are slow on generic log processors for very good reasons. For example, a simple matching by regular expression
could be 50 times slower than [special glob pattern](https://github.com/gobwas/glob), and possibly allocates millions
of buffers in memory heap which then need more time to be freed. The boundary crossing scripting interface is another
bottleneck, with marshalling and unmarshalling for each records that could cost more than the script execution itself.

Without any of such generic transforms and parsers, everything needs to be done in code, or blocks of code that you
can assemble together - which is essentially what this log agent does, a base and blocks of code for you to build high
performance log processors - if you need that kind of performance. The design is pluggable and the program is largely
configurable, but you're going to run into situations which can only be solved by reading or writing new code.


## Features

- Input: RFC 5424 Syslog protocol via TCP (with experimental multiline support)
- Transforms: field extraction and creations, drop, truncate, if/switch, email redaction
- Buffering: hybrid disk+memory buffering - compressed and only use disk when necessary
- Output: Fluentd Forward protocol, both compressed and uncompressed. Single output only.
- Metrics: Prometheus metrics to count logs and log size by key fields
- Parallelization: by key fields and by fixed pipeline numbers (WIP). Log order is preserved.

Dynamic fields are not supported - All fields must be known in configuration because they're packed in arrays without
hashmap lookup.

"tags" or similar concept doesn't exist here. Instead there are "if" and "switch-case" matching field values.

See the [sample configurations](testdata/config_sample.yml) for full features.


## Build

Requires [gotils](https://github.com/relex/gotils) which provides build tools

```bash
make
make test
```

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

## Runtime diagnosis

Prometheus listener address (default *:9335*) exposes go's `debug/pprof` in addition to metrics,
which can dump goroutine stacks.

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

- `` (empty): default to forward to fluentd defined in config.
  Chunks may not be fully sent upon shutdown and unsent chunks would be saved for next run.
- `null`: no output
- `.../%s`: create fluentd forward message files each of individual chunks at the path (`%s` as chunk ID). The dir must exist.
- `.../%s.json`: create JSON files each of individual chunks at the path (`%s` as chunk ID). The dir must exist.

fluentd forward message files can be examined by [fluentlibtool](https://github.com/relex/fluentlib)

## Internals

See [DESIGN](DESIGN.md)

#### Dependencies

- [klauspost' compress library](github.com/klauspost/compress) for fast gzip compression which is absolutely critical in the agent: benchmark & upgrade for new releasees
- [YAML v3](gopkg.in/yaml.v3) required for custom tags in configuration. `KnownFields` is still not working and it cannot check non-existent or misspelled properties.

#### Go upgrades

Some code is based on the internal behaviors of go runtime and marked as `GO_INTERNAL` in comments:

- util/syncmutex.go: `TryLockMutex()` (try to lock without waiting)
- util/syncwaitgroup.go: `PeekWaitGroup()` (peek the count number)
- util/strings.go: `StringFromBytes()` (make string from mutable bytes without copying)

They need to re-checked for any major upgrade of go for compatibility

## Authors

Special thanks to _Henrik Sjöström_ for his guiding on Go optimization, integration testing and invaluable suggestions
on performant design.
