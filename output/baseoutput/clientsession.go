package baseoutput

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util"
)

// clientSession represents a session bound to one forwarding connection
type clientSession struct {
	logger       logger.Logger
	inputChannel <-chan base.LogChunk
	inputClosed  channels.Awaitable
	onChunkAcked func(chunk base.LogChunk)
	metrics      clientMetrics
	conn         ClientConnection
	leftovers    chan base.LogChunk        // unprocessed chunks from previous session(s)
	lastChunk    *base.LogChunk            // last chunk in processing (to be added to leftovers if not completed)
	ackerChan    chan base.LogChunk        // channel to pass chunks for acknowledger (wait for ACK and delete), close to end acknowledger
	ackerAbort   *channels.SignalAwaitable // signal to abort acknowledger immediately
	ackerEnded   *channels.SignalAwaitable // signal that acknowledger has ended
	unacked      atomic.Value              // *[]base.LogChunk, un-ACK'ed chunks set when acknowledger quits (to be resent in next session)
}

type reconnectPolicy string
type acknowledgerEnding string

const (
	reconnectWithDelay reconnectPolicy    = "reconnectWithDelay"
	reconnect          reconnectPolicy    = "reconnect"
	noReconnect        reconnectPolicy    = "noReconnect"
	waitPendingChunks  acknowledgerEnding = "waitPendingChunks"
	endImmediately     acknowledgerEnding = "endImmediately"
)

func newClientSession(client *ClientWorker, conn ClientConnection, leftovers chan base.LogChunk) *clientSession {
	return &clientSession{
		logger:       conn.Logger().WithField(defs.LabelPart, "session"),
		inputChannel: client.inputChannel,
		inputClosed:  client.inputClosed,
		onChunkAcked: client.onChunkAcked,
		metrics:      client.metrics,
		conn:         conn,
		leftovers:    leftovers,
		lastChunk:    nil,
		ackerChan:    make(chan base.LogChunk, defs.ForwarderMaxPendingChunksForAck),
		ackerAbort:   channels.NewSignalAwaitable(),
		ackerEnded:   channels.NewSignalAwaitable(),
		unacked:      atomic.Value{},
	}
}

func (session *clientSession) run(maxDuration time.Duration) (chan base.LogChunk, reconnectPolicy) {
	go session.runAcknowledger()

	// send leftovers from previous sessions
	session.logger.Infof("begin recovery stage with leftovers=%d", len(session.leftovers))

REPLAY_LEFTOVERS:
	for {
		var chunk base.LogChunk
		var ok bool

		// get the next chunk to forward
		select {
		case <-session.inputClosed.Channel():
			session.logger.Infof("stop requested (recovery stage)")
			return session.collectLeftovers(endImmediately), noReconnect

		case chunk, ok = <-session.leftovers:
			if !ok {
				session.logger.Errorf("BUG: aborted due to leftover channel closure. stack=%s", util.Stack())
				return nil, noReconnect
			}
			session.metrics.OnLeftoverPopped(chunk)
			session.logger.Debugf("resending: %v", &chunk)
			session.lastChunk = &chunk

		default:
			// break as soon as there is no leftover to process
			break REPLAY_LEFTOVERS
		}

		// forward chunk
		continueSession, netErr := session.sendChunk(chunk)
		switch {
		case netErr != nil:
			return session.collectLeftovers(endImmediately), reconnectWithDelay
		case !continueSession:
			return session.collectLeftovers(endImmediately), noReconnect
		default:
			session.lastChunk = nil
		}
	}

	maxSessionDurationSignal := time.After(maxDuration)

	session.logger.Infof("begin normal stage with queued=%d", len(session.inputChannel))
	for {
		var chunk base.LogChunk
		var ok bool

		// get the next chunk to forward
		select {
		case chunk, ok = <-session.inputChannel:
			if !ok {
				session.logger.Infof("stop requested (normal stage)")
				return session.collectLeftovers(endImmediately), noReconnect
			}
			session.logger.Debugf("received new: %v", &chunk)
			session.lastChunk = &chunk

		case <-maxSessionDurationSignal:
			session.logger.Info("max session duration reached, stopping to reconnect")
			return session.collectLeftovers(waitPendingChunks), reconnect

		case <-time.After(defs.ForwarderPingInterval): // send ping (keep-alive) if there is no new log
			if err := session.sendPing(); err != nil {
				return session.collectLeftovers(endImmediately), reconnectWithDelay
			}
			continue
		}

		// forward chunk
		continueSession, netErr := session.sendChunk(chunk)
		switch {
		case netErr != nil:
			return session.collectLeftovers(endImmediately), reconnectWithDelay
		case !continueSession:
			return session.collectLeftovers(endImmediately), noReconnect

		default:
			session.lastChunk = nil
		}
	}
}

func (session *clientSession) sendChunk(chunk base.LogChunk) (bool, error) {
	session.metrics.OnForwarding(chunk)
	session.logger.Debugf("forward chunk %s", chunk.String())
	timeout := defs.ForwarderBatchSendTimeoutBase + time.Duration(len(chunk.Data)/defs.ForwarderBatchSendMinimumSpeed)*time.Second
	if err := session.conn.SendChunk(chunk, time.Now().Add(timeout)); err != nil {
		session.logger.Warnf("failed to send: %s, %s", chunk.String(), err.Error())
		session.metrics.OnError(err)
		return true, err
	}

	// pass forwarded chunk to acknowledger
	select {
	case session.ackerChan <- chunk:
		break
	case <-session.inputClosed.Channel():
		session.logger.Infof("aborted before queueing chunk for ack due to stop request: %s", chunk.String())
		return false, nil
	case <-session.ackerEnded.Channel():
		// acknowledger terminated due to invalid server response, return true and error for reconnection
		err := fmt.Errorf("aborted before queueing chunk for ack due to termination of acknowledger: %s", chunk.String())
		session.logger.Info(err.Error())
		return true, err
	}
	session.metrics.OnForwarded(chunk)
	return true, nil
}

// sendPing sends a forward message of zero logs and no ID (no ACK) to report status to server
func (session *clientSession) sendPing() error {
	session.logger.Debugf("forward ping")
	if err := session.conn.SendPing(time.Now().Add(defs.ForwarderBatchSendTimeoutBase)); err != nil {
		session.logger.Warnf("failed to ping: %s", err.Error())
		session.metrics.OnError(err)
		return err
	}
	return nil
}

func (session *clientSession) collectLeftovers(ending acknowledgerEnding) chan base.LogChunk {
	// gather previous leftovers
	close(session.leftovers)
	fromPrevious := collectChunksFromChannel(session.leftovers)

	switch ending {
	case waitPendingChunks:
		session.logger.Info("stopping acknowledger (soft)")
		close(session.ackerChan)
	case endImmediately:
		session.logger.Info("stopping acknowledger (hard)")
		session.ackerAbort.Signal()
		close(session.ackerChan)
	default:
		session.logger.Panic("invalid ending type: ", ending)
	}

	if !session.ackerEnded.Wait(defs.ForwarderAckerStopTimeout) {
		session.logger.Errorf("BUG: timeout waiting for acknowledger to stop. stack=%s", util.Stack())
	}
	fromAckerChannel := collectChunksFromChannel(session.ackerChan)

	// gather pendings chunks left in runAcknowledger's pendingChunksByID
	var fromAckerPending []base.LogChunk
	if pendingListPtr := session.unacked.Load().(*[]base.LogChunk); pendingListPtr != nil {
		fromAckerPending = *pendingListPtr
	} else {
		session.logger.Errorf("BUG: failed to get un-ACK'ed chunks from acknowledger. stack=%s", util.Stack())
	}

	// merge
	newLeftovers := make([]base.LogChunk, 0, len(fromPrevious)+len(fromAckerChannel)+len(fromAckerPending)+1)
	newLeftovers = append(newLeftovers, fromPrevious...)
	newLeftovers = append(newLeftovers, fromAckerChannel...)
	newLeftovers = append(newLeftovers, fromAckerPending...)

	// add the lastChunk in the middle of processing by clientSession.run
	inproc := 0
	if session.lastChunk != nil {
		newLeftovers = append(newLeftovers, *session.lastChunk)
		inproc++
	}
	session.metrics.OnSessionEnded(len(fromPrevious), len(fromAckerChannel)+len(fromAckerPending), len(newLeftovers))

	// remove duplicates
	newLeftoversChan := newLeftoverChannel(newLeftovers)
	session.logger.Infof("collected leftovers: prev(%d) + chan(%d) + unack(%d) + inproc(%d) = unique(%d)",
		len(fromPrevious), len(fromAckerChannel), len(fromAckerPending), inproc, len(newLeftoversChan))

	return newLeftoversChan
}

func (session *clientSession) runAcknowledger() {
	clogger := session.logger.WithField(defs.LabelPart, "session-acker")
	pendingChunksByID := make(map[string]base.LogChunk)
	defer func() {
		values := make([]base.LogChunk, 0, len(pendingChunksByID))
		for _, v := range pendingChunksByID {
			values = append(values, v)
		}
		session.unacked.Store(&values)
		session.ackerEnded.Signal()
	}()
	for {
		var nextChunk base.LogChunk

		// wait for next sent chunk from clientSession.run
		{
			select {
			case chunk, ok := <-session.ackerChan:
				if !ok {
					if len(pendingChunksByID) > 0 {
						clogger.Errorf("soft-stop requested while there are still pending chunks: %d", len(pendingChunksByID))
					} else {
						clogger.Info("soft-stop requested")
					}
					return
				}
				pendingChunksByID[chunk.ID] = chunk
				nextChunk = chunk
				clogger.Debugf("received pending chunk %s", chunk.ID)
			case <-session.ackerAbort.Channel():
				clogger.Info("stop requested, abort acknowledger")
				return
			}
		}

		// wait for next acknowledgement from upstream with timeout,
		// normally it should match the chunk we just received from ackerChan
		ackedChunkID, ackErr := session.conn.ReadChunkAck(time.Now().Add(defs.ForwarderBatchAckTimeout))
		if ackErr != nil {
			clogger.Warnf("failed to read ACK: %s", ackErr.Error())
			session.metrics.OnError(ackErr)
			session.conn.Close() // close both directions in case client=>server is still working
			return
		}

		clogger.Debugf("received ACK to chunk ID=%s", nextChunk.ID)

		// check the ID of the acknowledged chunk, not necessarily the same chunk received from clientSession.run
		if ackedChunkID != "" {
			if chunk, exists := pendingChunksByID[ackedChunkID]; exists {
				nextChunk = chunk
			} else {
				clogger.Errorf("received ACK to unknown chunk ID=%s", ackedChunkID)
				session.metrics.OnError(nil)
				continue
			}
		}

		// clean up the chunk we just processed
		delete(pendingChunksByID, nextChunk.ID)
		session.onChunkAcked(nextChunk)
		session.metrics.OnAcknowledged(nextChunk)
	}
}

// collectChunksFromChannel collect remaining chunks from a CLOSED channel
func collectChunksFromChannel(chunkChan chan base.LogChunk) []base.LogChunk {
	collected := make([]base.LogChunk, 0, len(chunkChan)+20)
	for c := range chunkChan {
		collected = append(collected, c)
	}
	return collected
}

// newLeftoverChannel creates a new channel filled with leftover chunks, sorted and deduplicated
func newLeftoverChannel(chunks []base.LogChunk) chan base.LogChunk {
	sort.Slice(chunks, func(i, j int) bool { return chunks[i].ID < chunks[j].ID })
	channel := make(chan base.LogChunk, len(chunks))
	lastChunkID := "" // to skip duplications
	for _, c := range chunks {
		if c.ID == lastChunkID {
			continue
		}
		channel <- c
		lastChunkID = c.ID
	}
	return channel
}
