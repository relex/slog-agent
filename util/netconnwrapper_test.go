package util

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNetConnWrapper(t *testing.T) {
	socket, err := net.Listen("tcp", "localhost:0")
	if !assert.Nil(t, err) {
		return
	}
	defer socket.Close()
	serverAddr := socket.Addr()
	go func() {
		client, cerr := net.Dial(serverAddr.Network(), serverAddr.String())
		if !assert.Nil(t, cerr) {
			return
		}
		defer client.Close()
		_, cerr = client.Write([]byte("Foo\n"))
		assert.Nil(t, cerr)
		time.Sleep(100 * time.Millisecond)
		_, cerr = client.Write([]byte("Bar\n"))
		assert.Nil(t, cerr)
		time.Sleep(100 * time.Millisecond)
		_, cerr = client.Write([]byte("Hello\n"))
		assert.Nil(t, cerr)
	}()
	server, err := socket.Accept()
	if !assert.Nil(t, err) {
		return
	}
	defer server.Close()
	reader := bufio.NewReaderSize(WrapNetConn(server, 40*time.Millisecond, 0), 1024)
	{
		ln, _, err := reader.ReadLine()
		assert.Nil(t, err)
		assert.Equal(t, "Foo", string(ln))
	}
	{
		// should timeout after 80ms
		_, _, err := reader.ReadLine()
		if !assert.True(t, IsNetworkTimeout(err)) {
			t.Error(err)
		}
	}
	{
		ln, _, err := reader.ReadLine()
		assert.Nil(t, err)
		assert.Equal(t, "Bar", string(ln))
	}
	{
		_, _, err := reader.ReadLine()
		if !assert.True(t, IsNetworkTimeout(err)) {
			t.Error(err)
		}
	}
	{
		ln, _, err := reader.ReadLine()
		assert.Nil(t, err)
		assert.Equal(t, "Hello", string(ln))
	}
}
