package util

import (
	"errors"
	"io"
	"net"
	"strings"
)

// IsNetworkClosed checks if the given error tells closing of network connection
func IsNetworkClosed(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Err.Error() == "use of closed network connection"
	}
	return false
}

// IsNetworkTimeout checks if the given error is network timeout
func IsNetworkTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// TrySetTCPReadBuffer attempts to set read buffer within the range given
func TrySetTCPReadBuffer(conn *net.TCPConn, max int, min int) (int, error) {
	var err error
	val := max
	for val >= min {
		err = conn.SetReadBuffer(val)
		if err == nil {
			return val, nil
		}
		if !strings.HasSuffix(err.Error(), "setsockopt: no buffer space available") {
			return -1, err
		}
		val /= 2
	}
	if val != min {
		err = conn.SetReadBuffer(min)
		if err == nil {
			return min, nil
		}
	}
	return -1, err
}
