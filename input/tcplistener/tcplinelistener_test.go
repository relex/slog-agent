package tcplistener

import (
	"net"
	"testing"
	"time"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base/btest"
	"github.com/relex/slog-agent/defs"
	"github.com/stretchr/testify/assert"
)

func TestTCPLineListener(t *testing.T) {
	const line1 = "<163>1 2019-08-15T15:50:46.866915+03:00 local my-app 123 fn - Something"
	const line2 = "<163>1 2019-08-16T15:50:46.866915+03:00 local my-app 123 fn - Something else"
	const line3 = "<163>1 2019-08-16T15:50:46.866915+03:00 local my-app 123 fn - End"
	const addrParam = "localhost:0"
	rlogger := logger.WithField("test", t.Name())
	stop := channels.NewSignalAwaitable()
	recv, out := btest.NewLogMessageAggregator(rlogger)
	lsnr, addr, err := NewTCPLineListener(rlogger, addrParam, testLine, recv, stop)
	assert.Nil(t, err)
	assert.NotEqual(t, addrParam, addr)
	lsnr.Start()
	conn, err := net.Dial("tcp", addr)
	if !assert.Nil(t, err) {
		t.Error(err)
	}
	_, err = conn.Write([]byte(line1 + "\n"))
	assert.Nil(t, err)
	_, err = conn.Write([]byte(line2 + "\n"))
	assert.Nil(t, err)
	assert.Equal(t, line1, readCh(out))
	assert.Equal(t, line2, readCh(out))
	_, err = conn.Write([]byte(line3)) // no newline end - close should force flushing
	assert.Nil(t, err)
	assert.Nil(t, conn.Close())
	assert.Equal(t, line3, readCh(out))
	stop.Signal()
	assert.True(t, lsnr.Stopped().Wait(defs.TestReadTimeout))
}

func TestTCPLineListenerEnd(t *testing.T) {
	const line1 = "<163>1 2019-08-15T15:50:46.866915+03:00 local my-app 123 fn - abc"
	const line2 = "<163>1 2019-08-15T15:50:46.866915+03:00 local my-app 123 fn - def"
	rlogger := logger.WithField("test", t.Name())
	stop := channels.NewSignalAwaitable()
	recv, out := btest.NewLogMessageAggregator(rlogger)
	lsnr, addr, _ := NewTCPLineListener(rlogger, "localhost:0", testLine, recv, stop)
	lsnr.Start()
	conn, _ := net.Dial("tcp", addr)
	_, err := conn.Write([]byte(line1 + "\n"))
	assert.Nil(t, err)
	assert.Equal(t, line1, readCh(out))
	_, err = conn.Write([]byte(line2)) // no newline end - close should force flushing
	assert.Nil(t, err)
	time.Sleep(500 * time.Millisecond)
	stop.Signal()
	assert.True(t, lsnr.Stopped().Wait(defs.TestReadTimeout))
	assert.Nil(t, conn.Close())
	assert.Equal(t, line2, readCh(out))
}

func TestTCPLineListenerMultiRead(t *testing.T) {
	oldBufferSize := defs.ListenerLineBufferSize
	defs.ListenerLineBufferSize = 10
	const line = "<163>1 2019-08-15T15:50:46.866915+03:00 local my-app 123 fn - Something"
	rlogger := logger.WithField("test", t.Name())
	stop := channels.NewSignalAwaitable()
	recv, out := btest.NewLogMessageAggregator(rlogger)
	lsnr, addr, _ := NewTCPLineListener(rlogger, "localhost:0", testLine, recv, stop)
	lsnr.Start()
	conn, _ := net.Dial("tcp", addr)
	_, err := conn.Write([]byte(line)) // no newline end - close should force flushing
	assert.Nil(t, err)
	assert.Nil(t, conn.Close())
	assert.Equal(t, line, readCh(out))
	stop.Signal()
	assert.True(t, lsnr.Stopped().Wait(defs.TestReadTimeout))
	defs.ListenerLineBufferSize = oldBufferSize
}

func testLine(ln []byte) bool {
	return true
}

func readCh(ch <-chan string) string {
	select {
	case log := <-ch:
		return log
	case <-time.After(defs.TestReadTimeout):
		return "<timeout>"
	}
}
