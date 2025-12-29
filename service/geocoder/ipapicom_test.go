//go:build localonly

package geocoder

import (
	"testing"
)

func TestIPAPICOM(t *testing.T) {
	encoder := NewIPAPICOM("")
	testGeocodeNetwork(t, encoder)
}
