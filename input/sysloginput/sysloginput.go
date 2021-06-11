// Package sysloginput provides an input source for Syslog (RFC 5424) protocol via TCP
//
// Multi-line (malformed) input is supported by recognizing syslog headers. Due to multi-line support, all input
// records are delayed until the arrival of the next record or flush timeout.
package sysloginput

import (
	"fmt"
	"net"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/input/syslogparser"
	"github.com/relex/slog-agent/input/syslogprotocol"
	"github.com/relex/slog-agent/input/tcplistener"
	"github.com/relex/slog-agent/transform"
)

// TODO: replace the protocol with something reliable such as RELP or MQ to prevent loss during restarts.

// Config provides configuration for SyslogInput
type Config struct {
	bconfig.Header `yaml:",inline"`
	Address        string                             `yaml:"address"`      // network address, e.g. "localhost:514". Empty host or port means any.
	LevelMapping   []string                           `yaml:"levelMapping"` // map syslog severity number to level name
	Extractions    []bconfig.LogTransformConfigHolder `yaml:"extractions"`  // transforms to run immediately after parser
}

type input struct {
	listener base.LogListener
	address  string
}

func init() {
	transform.Register() // for Extractions
}

// NewInput creates a SyslogInput and starts the network listener
func (cfg *Config) NewInput(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
	logBufferReceiver base.MultiChannelBufferReceiver, metricFactory *base.MetricFactory,
	stopRequest channels.Awaitable) (base.LogInput, error) {

	if len(cfg.LevelMapping) == 0 {
		return nil, fmt.Errorf(".levelMapping is empty")
	}
	if len(cfg.Extractions) == 0 {
		return nil, fmt.Errorf(".extractions is empty")
	}

	inputLogger := logger.WithField(defs.LabelComponent, "SyslogInput")

	// createParser is for each of incoming connection to create their own parser instance (which contains buffer/cache)
	createParser := func(parentLogger logger.Logger, inputCounter *base.LogInputCounter) base.LogParser {
		parser, err := syslogparser.NewParser(parentLogger, allocator, schema, cfg.LevelMapping, inputCounter)
		if err != nil {
			parentLogger.Panic("failed to create parser: ", err)
		}
		extractionTransforms := bsupport.NewTransformsFromConfig(cfg.Extractions, schema, inputLogger, inputCounter)
		return newCompositeParser(parser, extractionTransforms, allocator)
	}

	inputMetricFactory := metricFactory.NewSubFactory("input_", []string{"protocol"}, []string{"syslog"})

	rawMessageReceiver := bsupport.NewLogParsingReceiver(inputLogger, createParser, logBufferReceiver, inputMetricFactory)

	lsnr, addr, err := tcplistener.NewTCPLineListener(inputLogger, cfg.Address, syslogprotocol.TestRecordStart, rawMessageReceiver, stopRequest)
	if err != nil {
		return nil, err
	}

	return &input{
		listener: lsnr,
		address:  addr,
	}, nil
}

// NewParser creates a parser for test pipeline
func (cfg *Config) NewParser(parentLogger logger.Logger, allocator *base.LogAllocator, schema base.LogSchema,
	inputCounter *base.LogInputCounter) (base.LogParser, error) {

	if len(cfg.LevelMapping) == 0 {
		return nil, fmt.Errorf(".levelMapping is empty")
	}
	if len(cfg.Extractions) == 0 {
		return nil, fmt.Errorf(".extractions is empty")
	}

	slogger := parentLogger.WithField(defs.LabelComponent, "SyslogInput")

	return newCompositeParser(
		syslogparser.MustNewParser(slogger, allocator, schema, cfg.LevelMapping, inputCounter),
		bsupport.NewTransformsFromConfig(cfg.Extractions, schema, slogger, inputCounter),
		allocator,
	), nil
}

// VerifyConfig checks configuration
func (cfg *Config) VerifyConfig(schema base.LogSchema) error {
	if _, _, err := net.SplitHostPort(cfg.Address); err != nil {
		return fmt.Errorf(".address has invalid format: %w", err)
	}
	if len(cfg.Extractions) == 0 {
		return fmt.Errorf(".extractions is empty")
	}
	if err := bsupport.VerifyTransformConfigs(cfg.Extractions, schema, ".extractions"); err != nil {
		return err
	}
	return nil
}

func (in *input) Address() string {
	return in.address
}

func (in *input) Stopped() channels.Awaitable {
	return in.listener.Stopped()
}

func (in *input) Launch() {
	in.listener.Launch()
}
