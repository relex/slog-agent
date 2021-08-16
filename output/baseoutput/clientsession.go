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
)

// clientSession represents a session bound to one forwarding connection
type clientSession struct {
	logger       logger.Logger
	inputChannel <-chan base.LogChunk
	inputClosed  channels.Awaitable
	onChunkAcked func(chunk base.LogChunk)
	metrics      clientMetrics
	conn         ClientConnection
	leftovers    chan base.LogChunk        // unprocessed buffers from previous session(s)
	lastChunk    *base.LogChunk            // last buffer in processing (to be added to leftovers if not completed)
	ackerChan    chan base.LogChunk        // channel to pass buffers for acknowledger (wait for ACK and delete), close to end acknowledger
	ackerQuit    *channels.SignalAwaitable // channel for acknowledger to signal its end
	unacked      atomic.Value              // *[]base.LogChunk, un-ACK'ed buffers set when acknowledger quits (to be resent in next session)
}

func newClientSession(client *ClientWorker, conn ClientConnection, leftovers chan base.LogChunk) *clientSession {
	// set write buffer to 10MB
	return &clientSession{
		logger:       conn.Logger().WithField(defs.LabelPart, "session"),
		inputChannel: client.inputChannel,
		inputClosed:  client.inputClosed,
		onChunkAcked: client.onChunkAcked,
		metrics:      client.metrics,
		conn:         conn,
		leftovers:    leftovers,
		lastChunk:    nil,
		ackerChan:    make(chan base.LogChunk, defs.ForwarderMaxBuffersForAck),
		ackerQuit:    channels.NewSignalAwaitable(),
		unacked:      atomic.Value{},
	}
}

func (session *clientSession) run() (chan base.LogChunk, bool) {
	go session.runAcknowledger()
	// send leftovers from previous sessions
	session.logger.Infof("begin recovery stage with leftovers=%d", len(session.leftovers))
REPLAY_LEFTOVERS:
	for {
		var chunk base.LogChunk
		var ok bool
		// in
		select {
		case <-session.inputClosed.Channel():
			session.logger.Infof("stop requested (recovery stage)")
			return session.collectLeftovers(), false
		case chunk, ok = <-session.leftovers:
			if !ok {
				session.logger.Errorf("BUG: aborted due to leftover channel closure")
				return nil, false
			}
			session.metrics.OnLeftoverPopped(chunk)
			session.logger.Debugf("resending: %v", &chunk)
			session.lastChunk = &chunk
		default:
			// break as soon as there is no leftover to process
			break REPLAY_LEFTOVERS
		}
		// out
		continueSession, netErr := session.sendChunk(chunk)
		switch {
		case netErr != nil:
			return session.collectLeftovers(), true
		case !continueSession:
			return session.collectLeftovers(), false
		default:
			session.lastChunk = nil
		}
	}
	// send buffers from the channel
	session.logger.Infof("begin normal stage with queued=%d", len(session.inputChannel))
	for {
		var chunk base.LogChunk
		var ok bool
		// in
		select {
		case chunk, ok = <-session.inputChannel:
			if !ok {
				session.logger.Infof("stop requested (normal stage)")
				return session.collectLeftovers(), false
			}
			session.logger.Debugf("received new: %v", &chunk)
			session.lastChunk = &chunk
		case <-time.After(defs.ForwarderPingInterval):
			if err := session.sendPing(); err != nil {
				return session.collectLeftovers(), true
			}
			continue
		}
		// out
		continueSession, netErr := session.sendChunk(chunk)
		switch {
		case netErr != nil:
			return session.collectLeftovers(), true
		case !continueSession:
			return session.collectLeftovers(), false
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
		session.metrics.IncrementNetworkErrors()
		return true, err
	}
	select {
	case session.ackerChan <- chunk:
		break
	case <-session.inputClosed.Channel():
		session.logger.Infof("aborted before queueing ack buf due to stop request: %s", chunk.String())
		return false, nil
	case <-session.ackerQuit.Channel():
		// acknowledger terminated due to invalid server response, return true and error for reconnection
		err := fmt.Errorf("aborted before queueing ack buf due to termination of acknowledger: %s", chunk.String())
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
		session.metrics.IncrementNetworkErrors()
		return err
	}
	return nil
}

func (session *clientSession) collectLeftovers() chan base.LogChunk {
	// gather unhandled previous leftovers
	close(session.leftovers)
	fromPrevious := channelToBuffer(session.leftovers)
	// stop acknowledger and gather unread chunks from its input channel
	session.logger.Info("stopping acknowledger")
	close(session.ackerChan)
	if !session.ackerQuit.Wait(defs.ForwarderAckerStopTimeout) {
		session.logger.Error("BUG: timeout waiting for acknowledger to stop")
	}
	fromAckerChannel := channelToBuffer(session.ackerChan)
	// gather un-ACK'ed chunks set by runAcknowledger (pendingAckBufMap)
	var fromAckerUnacked []base.LogChunk
	if unackedPtr := session.unacked.Load().(*[]base.LogChunk); unackedPtr != nil {
		fromAckerUnacked = *unackedPtr
	} else {
		session.logger.Error("BUG: failed to get un-ACK'ed chunks from acknowledger")
	}
	// merge
	newLeftovers := make([]base.LogChunk, 0, len(fromPrevious)+len(fromAckerChannel)+len(fromAckerUnacked)+1)
	newLeftovers = append(newLeftovers, fromPrevious...)
	newLeftovers = append(newLeftovers, fromAckerChannel...)
	newLeftovers = append(newLeftovers, fromAckerUnacked...)
	// last in sending
	inproc := 0
	if session.lastChunk != nil {
		newLeftovers = append(newLeftovers, *session.lastChunk)
		inproc++
	}
	session.metrics.OnSessionEnded(len(fromPrevious), len(fromAckerChannel)+len(fromAckerUnacked), len(newLeftovers))
	// remove duplicates
	newLeftoversChan := bufferToChannel(newLeftovers)
	session.logger.Infof("collected leftovers: prev(%d) + chan(%d) + unack(%d) + inproc(%d) = unique(%d)",
		len(fromPrevious), len(fromAckerChannel), len(fromAckerUnacked), inproc, len(newLeftoversChan))
	return newLeftoversChan
}

func (session *clientSession) runAcknowledger() {
	clogger := session.logger.WithField(defs.LabelPart, "session-acker")
	pendingAckBufMap := make(map[string]base.LogChunk)
	defer func() {
		values := make([]base.LogChunk, 0, len(pendingAckBufMap))
		for _, v := range pendingAckBufMap {
			values = append(values, v)
		}
		session.unacked.Store(&values)
		session.ackerQuit.Signal()
	}()
	for {
		var nextChunk base.LogChunk

		// wait for a buffer for ACK
		{
			chunk, ok := <-session.ackerChan
			if !ok {
				clogger.Infof("stop requested")
				return
			}
			pendingAckBufMap[chunk.ID] = chunk
			nextChunk = chunk
			clogger.Debugf("received pending chunk %s", chunk.ID)
		}

		// wait for ACK from upstream
		ackedChunkID, ackErr := session.conn.ReadChunkAck(time.Now().Add(defs.ForwarderBatchAckTimeout))
		if ackErr != nil {
			clogger.Warnf("failed to read ACK: %s", ackErr.Error())
			session.metrics.IncrementNetworkErrors()
			session.conn.Close() // close both directions in case client=>server is still working
			return
		}

		clogger.Debugf("received ACK to chunk ID=%s", nextChunk.ID)

		// check the ACK returned from server, not necessarily for the buffer received above
		if ackedChunkID != "" {
			if chunk, exists := pendingAckBufMap[ackedChunkID]; exists {
				nextChunk = chunk
			} else {
				clogger.Errorf("received ACK to unknown chunk ID=%s", ackedChunkID)
				continue
			}
		}

		// clean up
		delete(pendingAckBufMap, nextChunk.ID)
		session.onChunkAcked(nextChunk)
		session.metrics.OnAcknowledged(nextChunk)
	}
}

func channelToBuffer(bufChan chan base.LogChunk) []base.LogChunk {
	collected := make([]base.LogChunk, 0, len(bufChan)+20)
	for c := range bufChan {
		collected = append(collected, c)
	}
	return collected
}

func bufferToChannel(buffers []base.LogChunk) chan base.LogChunk {
	sort.Slice(buffers, func(i, j int) bool { return buffers[i].ID < buffers[j].ID })
	channel := make(chan base.LogChunk, len(buffers))
	lastChunkID := "" // to skip duplications
	for _, b := range buffers {
		if b.ID == lastChunkID {
			continue
		}
		channel <- b
		lastChunkID = b.ID
	}
	return channel
}
