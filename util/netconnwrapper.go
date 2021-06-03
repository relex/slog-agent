package util

import (
	"net"
	"time"
)

// NetConnWrapper wraps a connection with with timeouts updated infrequently in trade of accuracy
// The real timeouts could be anything from specified timeout values to double of them
type NetConnWrapper struct {
	conn            net.Conn
	readTimeoutMin  time.Duration
	readTimeoutMax  time.Duration
	readDeadline    time.Time
	writeTimeoutMin time.Duration
	writeTimeoutMax time.Duration
	writeDeadline   time.Time
}

// WrapNetConn creates a NetConnWrapper for given network connection
func WrapNetConn(conn net.Conn, readTimeout time.Duration, writeTimeout time.Duration) *NetConnWrapper {
	return &NetConnWrapper{
		conn:            conn,
		readTimeoutMin:  readTimeout,
		readTimeoutMax:  readTimeout * 2,
		readDeadline:    time.Time{},
		writeTimeoutMin: writeTimeout,
		writeTimeoutMax: writeTimeout * 2,
		writeDeadline:   time.Time{},
	}
}

// ReadDeadline returns the current read deadline
func (cw *NetConnWrapper) ReadDeadline() time.Time {
	return cw.readDeadline
}

func (cw *NetConnWrapper) Read(p []byte) (n int, err error) {
	if cw.readTimeoutMin > 0 {
		now := time.Now()
		if cw.readDeadline.Sub(now) < cw.readTimeoutMin {
			nextDeadline := now.Add(cw.readTimeoutMax)
			if err := cw.conn.SetReadDeadline(nextDeadline); err != nil {
				return 0, err
			}
			cw.readDeadline = nextDeadline
		}
	}
	return cw.conn.Read(p)
}

// WriteDeadline returns the current write deadline
func (cw *NetConnWrapper) WriteDeadline() time.Time {
	return cw.writeDeadline
}

func (cw *NetConnWrapper) Write(p []byte) (int, error) {
	if cw.writeTimeoutMin > 0 {
		now := time.Now()
		if cw.writeDeadline.Sub(now) < cw.writeTimeoutMin {
			nextDeadline := now.Add(cw.writeTimeoutMax)
			if err := cw.conn.SetWriteDeadline(nextDeadline); err != nil {
				return 0, err
			}
			cw.writeDeadline = nextDeadline
		}
	}
	return cw.conn.Write(p)
}
