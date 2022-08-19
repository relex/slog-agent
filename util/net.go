package util

import (
	"errors"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/relex/gotils/logger"
)

// GetFDFromTCPConnOrPanic tries to get socket FD from the given connection or panic
//
// The connection must have been established first
func GetFDFromTCPConnOrPanic(conn *net.TCPConn) uintptr {
	fd, err := GetFDFromTCPConn(conn)
	if err != nil {
		logger.WithFields(logger.Fields{
			"local":  conn.LocalAddr().String(),
			"remote": conn.RemoteAddr().String(),
		}).Panic("failed to get FD from TCP connection: ", err)
	}
	return fd
}

// GetFDFromTCPConn reads socket FD from the given connection
func GetFDFromTCPConn(conn *net.TCPConn) (uintptr, error) {
	rawConn, connErr := conn.SyscallConn()
	if connErr != nil {
		return 0, connErr
	}

	var value uintptr
	fdErr := rawConn.Control(func(fd uintptr) {
		value = fd
	})
	return value, fdErr
}

// IsNetworkClosed checks if the given error tells closing of network connection
func IsNetworkClosed(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Err.Error() == "use of closed network connection"
	}
	return false
}

// IsNetworkError checks whether the given error should be considered a network issue,
// as opposed to e.g. expired certificate or permission denied
//
// DO NOT use net.Error directly, as we may need to check for other error types in future
func IsNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
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
