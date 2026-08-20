//go:build localonly

package geocoder

import (
	"testing"
)

// TestIPAPICOM exercises the ip-api.com geocoder against the shared network suite
func TestIPAPICOM(t *testing.T) {
	encoder := NewIPAPICOM("")
	testGeocodeNetwork(t, encoder)
}
