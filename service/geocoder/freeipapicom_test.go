//go:build localonly

package geocoder

import (
	"testing"
)

func TestFREEIPAPICOM(t *testing.T) {
	encoder := NewFREEIPAPICOM("")
	testGeocodeNetwork(t, encoder)
}
