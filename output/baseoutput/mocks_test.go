package baseoutput

import (
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
)

type clientMockEnv struct {
	InputChannel chan base.LogChunk
	InputClosed  *channels.SignalAwaitable

	NewConnection    EstablishConnectionFunc
	OnChunkConsumed  func(base.LogChunk)
	OnChunkLeftover  func(base.LogChunk)
	OnClientFinished func()

	ConnSendChunk    func(chunk base.LogChunk, deadline time.Time) error
	ConnSendPing     func(deadline time.Time) error
	ConnReadChunkAck func(deadline time.Time) (string, error)
	ConnClose        func()

	ClientWorker base.ChunkConsumer

	SentPingDeadlines   chan time.Time
	SentChunks          chan base.LogChunk
	AckedChunkDeadlines chan time.Time
	ConsumedChunks      chan base.LogChunk
}

func newClientMockEnv() *clientMockEnv {
	const bufSize = 5000

	env := &clientMockEnv{}
	env.InputChannel = make(chan base.LogChunk, 100)
	env.InputClosed = channels.NewSignalAwaitable()

	env.NewConnection = env.DefaultNewConnection
	env.OnChunkConsumed = env.DefaultOnChunkConsumed
	env.OnChunkLeftover = env.DefaultOnChunkLeftover
	env.OnClientFinished = env.DefaultOnClientFinished

	env.ConnSendChunk = env.DefaultConnSendChunk
	env.ConnSendPing = env.DefaultConnSendPing
	env.ConnReadChunkAck = env.DefaultConnReadChunkAck
	env.ConnClose = env.DefaultConnClose

	env.ClientWorker = NewClientWorker(
		logger.WithField(defs.LabelComponent, "MockClientWorker"),
		base.ChunkConsumerArgs{
			InputChannel:    env.InputChannel,
			InputClosed:     env.InputClosed,
			OnChunkConsumed: func(chunk base.LogChunk) { env.OnChunkConsumed(chunk) },
			OnChunkLeftover: func(chunk base.LogChunk) { env.OnChunkLeftover(chunk) },
			OnFinished:      func() { env.OnClientFinished() },
		},
		promreg.NewMetricFactory("testclientworker_", nil, nil),
		func() (ClosableClientConnection, error) { return env.NewConnection() },
		10*time.Second,
	)

	env.SentPingDeadlines = make(chan time.Time, bufSize)
	env.SentChunks = make(chan base.LogChunk, bufSize)
	env.AckedChunkDeadlines = make(chan time.Time, bufSize)
	env.ConsumedChunks = make(chan base.LogChunk, bufSize)

	return env
}

func (env *clientMockEnv) LaunchSession() (channels.Awaitable, *chan base.LogChunk, *reconnectPolicy) {
	mockConn, _ := env.NewConnection()
	sess := newClientSession(env.ClientWorker.(*ClientWorker), mockConn)

	sessionEnded := channels.NewSignalAwaitable()

	var leftovers chan base.LogChunk
	var policy reconnectPolicy
	go func() {
		leftovers, policy = sess.Run(make(chan base.LogChunk), 10*time.Second)
		close(leftovers)
		sessionEnded.Signal()
	}()

	return sessionEnded, &leftovers, &policy
}

func (env *clientMockEnv) DefaultNewConnection() (ClosableClientConnection, error) {
	return &clientMockConn{
		env:    env,
		logger: logger.WithField(defs.LabelComponent, "MockClientConnection"),
	}, nil
}

func (env *clientMockEnv) DefaultOnChunkConsumed(chunk base.LogChunk) {
	env.ConsumedChunks <- chunk
}

func (env *clientMockEnv) DefaultOnChunkLeftover(chunk base.LogChunk) {
}

func (env *clientMockEnv) DefaultOnClientFinished() {
}

func (env *clientMockEnv) DefaultConnSendChunk(chunk base.LogChunk, deadline time.Time) error {
	env.SentChunks <- chunk
	return nil
}

func (env *clientMockEnv) DefaultConnSendPing(deadline time.Time) error {
	env.SentPingDeadlines <- deadline
	return nil
}

func (env *clientMockEnv) DefaultConnReadChunkAck(deadline time.Time) (string, error) {
	env.AckedChunkDeadlines <- deadline
	return "", nil
}

func (env *clientMockEnv) DefaultConnClose() {
}

type clientMockConn struct {
	env    *clientMockEnv
	logger logger.Logger
}

func (conn *clientMockConn) Logger() logger.Logger {
	return conn.logger
}

func (conn *clientMockConn) SendChunk(chunk base.LogChunk, deadline time.Time) error {
	return conn.env.ConnSendChunk(chunk, deadline)
}

func (conn *clientMockConn) SendPing(deadline time.Time) error {
	return conn.env.ConnSendPing(deadline)
}

func (conn *clientMockConn) ReadChunkAck(deadline time.Time) (string, error) {
	return conn.env.ConnReadChunkAck(deadline)
}

func (conn *clientMockConn) Close() {
	conn.env.ConnClose()
}
