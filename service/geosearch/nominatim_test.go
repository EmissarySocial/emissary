//go:build localonly

package geosearch

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestNominatim(t *testing.T) {

	searchFunc := Nominatim("", "Emissary Test Suite", "/test/nominatim")
	result, err := searchFunc("1701 Wynkoop Street, Denver CO, 80202")

	spew.Dump(result)
	spew.Dump(err)
}

func TestNominatim_Snooze(t *testing.T) {

	searchFunc := Nominatim("", "Emissary Test Suite", "/test/nominatim")
	result, err := searchFunc("Snooze, Denver CO")

	spew.Dump(result)
	spew.Dump(err)
}
