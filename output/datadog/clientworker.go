package datadog

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/output/baseoutput"
)

type clientWorker struct {
	logger  logger.Logger
	client  *http.Client
	request *http.Request
}

func NewClientWorker(parentLogger logger.Logger, args base.ChunkConsumerArgs, metricCreator promreg.MetricCreator, cfg UpstreamConfig) base.ChunkConsumer {
	clientLogger := parentLogger.WithField(defs.LabelComponent, "DatadogClient")

	rq, err := http.NewRequest(http.MethodPost, cfg.Address, nil)
	if err != nil {
		parentLogger.Panic(err)
	}
	rq.Header.Add("Content-Encoding", "gzip")
	rq.Header.Add("Content-Type", "application/json")
	rq.Header.Add("DD-API-KEY", os.Getenv("DD_API_KEY"))

	worker := &clientWorker{
		logger:  parentLogger,
		client:  &http.Client{Timeout: cfg.HTTPTimeout},
		request: rq,
	}

	return baseoutput.NewClientWorker(
		clientLogger,
		args,
		metricCreator,
		func() (baseoutput.ClosableClientConnection, error) {
			return worker, nil
		},
		time.Hour*24*365, // no reconnects are required for an http connection
	)
}

func (worker *clientWorker) Logger() logger.Logger {
	return worker.logger
}

func (worker *clientWorker) SendChunk(chunk base.LogChunk, _ time.Time) error {
	worker.request.Body = io.NopCloser(bytes.NewReader(chunk.Data))
	defer func() { worker.request.Body = nil }()

	resp, err := worker.client.Do(worker.request)
	if err != nil {
		return fmt.Errorf("send chunk error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("couldn't read response body: %w", err)
		}
		return fmt.Errorf("got a status %d with body %s", resp.StatusCode, body)
	}
	return nil
}

func (worker *clientWorker) Close()                                          {}                 //nolint:revive
func (worker *clientWorker) SendPing(deadline time.Time) error               { return nil }     //nolint:revive
func (worker *clientWorker) ReadChunkAck(deadline time.Time) (string, error) { return "", nil } //nolint:revive
