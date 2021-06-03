AUTO_BUILD_VERSION ?= dev
GOPATH := $(shell go env GOPATH)
SOURCES_NONGEN := $(shell find . -name '*.go' -not -name '*_test.go' -not -name '*.gen.go')
SOURCES_GEN := $(shell find . -name '*.gen.go')
SOURCES_TPL := $(shell find . -name '*.tpl.go')
export LINT_EXHAUSTIVESTRUCT=Y

build: BUILD/slog-agent

include ${GOPATH}/opt/gotils/Common.mk

BUILD/slog-agent: Makefile go.mod $(SOURCES_NONGEN) $(SOURCES_GEN)
	GO_LDFLAGS="-X main.version=$(AUTO_BUILD_VERSION)" gotils-build.sh -o $@

$(SOURCES_GEN): $(SOURCES_TPL)
	go generate ./...

.PHONY: test-gen
test-gen: BUILD/slog-agent
	LOG_LEVEL=$${LOG_LEVEL:-warn} LOG_COLOR=$${LOG_COLOR:-Y} go test -timeout $${TEST_TIMEOUT:-10s} -v ./... -args gen

# Run all micro benchmarks
.PHONY: bench-micro
bench-micro: BUILD/slog-agent
	LOG_LEVEL=$${LOG_LEVEL:-warn} LOG_COLOR=$${LOG_COLOR:-Y} go test -run="^$$" -bench=. -benchmem ./...

# TODO: integrate benchmarks to CI

# Main benchmarks under different GOMAXPROCS, to check the cost of concurrent executions, that high GOMAXPROCS doesn't increase overall CPU time by much
.PHONY: bench-cpus
bench-cpus: BUILD/slog-agent
	# PIPELINE, 1 THREAD
	GOMAXPROCS=1 LOG_LEVEL=warn time BUILD/slog-agent --trace /tmp/pipeline-config-cpu-1.trace benchmark pipeline --input 'testdata/development/*-input.log' --repeat 100000 --config testdata/config_sample.yml
	# AGENT, 1 THREAD
	GOMAXPROCS=1 LOG_LEVEL=warn time BUILD/slog-agent --trace /tmp/agent-config-cpu-1.trace benchmark agent --input 'testdata/development/*-input.log' --repeat 100000 --config testdata/config_sample.yml
	# AGENT, cpuNum THREADs
	GOMAXPROCS= LOG_LEVEL=warn time BUILD/slog-agent --trace /tmp/agent-config-cpu-n.trace benchmark agent --input 'testdata/development/*-input.log' --repeat 100000 --config testdata/config_sample.yml
	# AGENT, 100 THREADs
	GOMAXPROCS=100 LOG_LEVEL=warn time BUILD/slog-agent --trace /tmp/agent-config-cpu-100.trace benchmark agent --input 'testdata/development/*-input.log' --repeat 100000 --config testdata/config_sample.yml

# Main benchmarks for different pipelines and agent types
.PHONY: bench-full
bench-full: BUILD/slog-agent
	LOG_LEVEL=warn time BUILD/slog-agent --cpuprofile /tmp/pipeline-config-cpu.profile benchmark pipeline --input 'testdata/development/*-input.log' --repeat 100000 --config testdata/config_sample.yml
	LOG_LEVEL=warn time BUILD/slog-agent --cpuprofile /tmp/agent-config-cpu.profile benchmark agent --input 'testdata/development/*-input.log' --repeat 100000 --config testdata/config_sample.yml
