package baseinput

import (
	"time"

	"github.com/relex/slog-agent/defs"
)

// ReadStringChannel reads from string channel or return timeout error
func ReadStringChannel(ch <-chan string) string {
	select {
	case log := <-ch:
		return log
	case <-time.After(defs.TestReadTimeout):
		return "<timeout>"
	}
}
