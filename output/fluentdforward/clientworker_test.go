package fluentdforward

import (
	"os"
	"testing"

	"github.com/relex/fluentlib/server"
	"github.com/relex/fluentlib/server/receivers"
	"github.com/relex/gotils/logger"
	"github.com/stretchr/testify/assert"
)

func TestOpenClientConnection(t *testing.T) {
	recv := receivers.NewMessageWriter(os.Stdout)

	t.Run("connect fails when protocols are different", func(t *testing.T) {
		srvCfg := server.Config{
			Address: "localhost:0",
			TLS:     false,
		}
		srv, srvAddr := server.LaunchServer(logger.WithField("test", t.Name()), srvCfg, recv)
		defer srv.Shutdown()

		_, err := openForwardConnection(logger.Root(), UpstreamConfig{
			Address: srvAddr.String(),
			TLS:     true, // attempt to request TLS handshake
		})
		assert.ErrorContains(t, err, "failed to connect:")
	})

	t.Run("login fails when secrets are different", func(t *testing.T) {
		srvCfg := server.Config{
			Address: "localhost:0",
			Secret:  "real pass",
			TLS:     false,
		}
		srv, srvAddr := server.LaunchServer(logger.WithField("test", t.Name()), srvCfg, recv)
		defer srv.Shutdown()

		_, err := openForwardConnection(logger.Root(), UpstreamConfig{
			Address: srvAddr.String(),
			TLS:     false,
			Secret:  "wrong pass",
		})
		assert.ErrorContains(t, err, "login rejected:")
	})
}
