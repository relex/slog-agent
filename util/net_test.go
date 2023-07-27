package util

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNet(t *testing.T) {
	lsnr, lerr := net.Listen("tcp", "localhost:0")
	assert.NoError(t, lerr)

	t.Log("listening " + lsnr.Addr().String())

	go func() {
		cconn, cerr := net.Dial("tcp", lsnr.Addr().String())
		assert.NoError(t, cerr)

		cconn.Close()
	}()

	sconn, serr := lsnr.Accept()
	assert.NoError(t, serr)

	t.Run("get FD", func(tt *testing.T) {
		fd := GetFDFromTCPConnOrPanic(sconn.(*net.TCPConn))
		assert.Greater(t, int(fd), 0)
		assert.Less(t, int(fd), 65536)
	})

	t.Run("set buffer", func(tt *testing.T) {
		maxSz := 1048576 * 16
		minSz := 1048576
		sz, err := TrySetTCPReadBuffer(sconn.(*net.TCPConn), maxSz, minSz)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, sz, minSz)
		assert.LessOrEqual(t, sz, maxSz)
	})

	t.Run("check error", func(tt *testing.T) {
		sconn.Close()
		_, err := sconn.Write([]byte("Hi"))
		if assert.Error(t, err) {
			assert.True(t, IsNetworkError(err))
			assert.True(t, IsNetworkClosed(err))
		}
	})
}
