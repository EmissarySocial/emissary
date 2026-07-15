package server

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service"
	"github.com/stretchr/testify/require"
)

func TestServerFactory(t *testing.T) {
	var factory service.ServerFactory = &Factory{}
	require.NotNil(t, factory)
}

// TestTestDatabaseConnection verifies the fail-fast behavior used by the setup console: an
// unreachable or malformed database connection must return an error QUICKLY, instead of
// hanging on the driver's default (~30s) server-selection timeout and leaving a broken
// domain persisted in the configuration file.
func TestTestDatabaseConnection(t *testing.T) {

	t.Run("Unreachable", func(t *testing.T) {
		// Port 1 has nothing listening; directConnection skips replica-set discovery so the
		// failure is bounded by our timeout rather than topology scanning.
		configuration := config.Domain{
			ConnectString: "mongodb://127.0.0.1:1/?directConnection=true",
			DatabaseName:  "emissary_test",
		}

		start := time.Now()
		err := testDatabaseConnection(configuration, 2*time.Second)
		elapsed := time.Since(start)

		require.Error(t, err)
		require.Less(t, elapsed, 10*time.Second, "connection check must fail fast, not hang")
	})

	t.Run("MalformedConnectString", func(t *testing.T) {
		configuration := config.Domain{
			ConnectString: "not-a-mongodb-uri",
			DatabaseName:  "emissary_test",
		}

		err := testDatabaseConnection(configuration, 2*time.Second)
		require.Error(t, err)
	})
}
