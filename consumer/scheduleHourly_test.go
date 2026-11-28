package consumer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHourlyOffsetSeconds confirms that each Domain's offset is stable, inside the window, and
// not shared by every Domain
func TestHourlyOffsetSeconds(t *testing.T) {

	hostnames := []string{"", "localhost", "example.com", "emissary.social", "bandwagon.fm", "xn--bcher-kva.example", "a.very.long.hostname.with.many.labels.example.org"}
	distinct := map[int]struct{}{}

	for _, hostname := range hostnames {

		offset := hourlyOffsetSeconds(hostname)

		require.GreaterOrEqual(t, offset, 0, "hostname %q", hostname)
		require.Less(t, offset, hourlyOffsetWindow, "hostname %q", hostname)
		require.Equal(t, offset, hourlyOffsetSeconds(hostname), "hostname %q gets the same offset every hour", hostname)

		distinct[offset] = struct{}{}
	}

	require.Len(t, distinct, len(hostnames), "these hostnames are spread apart")
}

// TestHourlyOffsetWindow confirms that a spread-out task starts inside its own hour
func TestHourlyOffsetWindow(t *testing.T) {
	require.Less(t, hourlyOffsetWindow, 3600)
}
