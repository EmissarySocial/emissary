//go:build localonly

package geocoder

import (
	"testing"
)

// TestFREEIPAPICOM exercises the freeipapi.com geocoder against the shared network suite
func TestFREEIPAPICOM(t *testing.T) {
	encoder := NewFREEIPAPICOM("")
	testGeocodeNetwork(t, encoder)
}
