package baseoutput

import (
	"fmt"
	"testing"
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestSessionShutdown(t *testing.T) {
	mockEnv := newClientMockEnv()
	sessionEnded, sessionLeftovers, sessionReconnectPolicy := mockEnv.LaunchSession()

	mockEnv.InputChannel <- base.LogChunk{ID: "c1", Data: []byte{'1'}, Saved: false}
	mockEnv.InputChannel <- base.LogChunk{ID: "c2", Data: []byte{'2'}, Saved: false}

	go func() {
		time.Sleep(100 * time.Millisecond)
		mockEnv.InputClosed.Signal()
		close(mockEnv.InputChannel)
	}()

	if !assert.True(t, sessionEnded.Wait(1*time.Second), "Session should end by itself due to input closing") {
		return
	}
	assert.Equal(t, noReconnect, *sessionReconnectPolicy)
	assert.Empty(t, util.CollectFromChannel(*sessionLeftovers))

	close(mockEnv.SentChunks)
	close(mockEnv.AckedChunkDeadlines)
	close(mockEnv.ConsumedChunks)

	assert.Equal(t, 2, len(mockEnv.SentChunks))

	// Note the followings are not guaranteed because acknowledger got a hard-stop and not necessarily received all ACKs.
	// There is no easy way to simulate this as soft-stop (graceful stopping) only happens by periodic reconnection or
	// signal, not commanded by caller.
	assert.Equal(t, 2, len(mockEnv.AckedChunkDeadlines))
	assert.Equal(t, 2, len(mockEnv.ConsumedChunks))
}

// TestRequestErrorAbortsAcknowledger verifies a session in half-close situation (1) is aborted immediately.
//
// This could happen when Fluentd refuses receving requests but continues to send ACKs normally.
func TestRequestErrorAbortsAcknowledger(t *testing.T) {
	abortSignal := channels.NewSignalAwaitable()

	mockEnv := newClientMockEnv()
	// Story:
	// 1. Send c1 => ok
	// 2. Send c2 => error (should abort conn) | Recv c1 => wait (should be aborted)
	mockEnv.ConnSendChunk = func(chunk base.LogChunk, deadline time.Time) error {
		if chunk.ID > "c1" {
			return fmt.Errorf("Connection broken (client->server)")
		}
		return mockEnv.DefaultConnSendChunk(chunk, deadline)
	}
	mockEnv.ConnReadChunkAck = func(deadline time.Time) (string, error) {
		dur := time.Until(deadline)
		logger.WithField(defs.LabelComponent, "MockClientAcknowledger").Info("Simulating non-responsive upstream: wait for", dur)
		if abortSignal.Wait(dur) {
			return "", fmt.Errorf("Connection aborted (server->client)")
		}
		return "", nil
	}
	mockEnv.ConnClose = func() {
		// simulate aborting connection to force ConnReadChunkAck to end (instead of waiting until timeout)
		abortSignal.Signal()
	}

	// Start
	sessionEnded, sessionLeftovers, sessionReconnectPolicy := mockEnv.LaunchSession()
	mockEnv.InputChannel <- base.LogChunk{ID: "c1", Data: []byte{'1'}, Saved: false}
	mockEnv.InputChannel <- base.LogChunk{ID: "c2", Data: []byte{'2'}, Saved: false}

	if !assert.True(t, sessionEnded.Wait(1*time.Second), "Session should end by itself due to network error") {
		return
	}

	assert.Equal(t, 2, len(*sessionLeftovers))
	assert.Equal(t, reconnectWithDelay, *sessionReconnectPolicy)
}

// TestResponseErrorAbortsConnection verifies a session in half-close situation (2) is aborted immediately.
//
// This could happen when Fluentd receives requests normally but the response side is already interrupted due to network issues.
func TestResponseErrorAbortsConnection(t *testing.T) {
	abortSignal := channels.NewSignalAwaitable()

	mockEnv := newClientMockEnv()
	// Story:
	// 1. Send c1 => ok
	// 2. Send c2 => wait (should be aborted) | Recv c1 => error (should abort conn)
	mockEnv.ConnSendChunk = func(chunk base.LogChunk, deadline time.Time) error {
		if chunk.ID > "c1" {
			dur := time.Until(deadline)
			logger.WithField(defs.LabelComponent, "MockClientSession").Info("Simulating non-responsive upstream: wait for ", dur)
			if abortSignal.Wait(dur) {
				return fmt.Errorf("Connection aborted (client->server)")
			}
		}
		return mockEnv.DefaultConnSendChunk(chunk, deadline)
	}
	mockEnv.ConnReadChunkAck = func(deadline time.Time) (string, error) {
		return "", fmt.Errorf("Connection broken (server->client)")
	}
	mockEnv.ConnClose = func() {
		// simulate aborting connection to force ConnSendChunk to end (instead of waiting until timeout)
		abortSignal.Signal()
	}

	// Start
	sessionEnded, sessionLeftovers, sessionReconnectPolicy := mockEnv.LaunchSession()
	mockEnv.InputChannel <- base.LogChunk{ID: "c1", Data: []byte{'1'}, Saved: false}
	mockEnv.InputChannel <- base.LogChunk{ID: "c2", Data: []byte{'2'}, Saved: false}

	if !assert.True(t, sessionEnded.Wait(1*time.Second), "Session should end by itself due to network error") {
		return
	}

	assert.Equal(t, 2, len(*sessionLeftovers))
	assert.Equal(t, reconnectWithDelay, *sessionReconnectPolicy)
}
