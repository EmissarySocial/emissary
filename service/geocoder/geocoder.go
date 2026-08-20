package geocoder

import (
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/sliceof"
)

// AddressGeocoder resolves a free-text address into structured coordinates
type AddressGeocoder interface {
	GeocodeAddress(string) (geo.Address, error)
}

// NetworkGeocoder resolves an IP address into approximate coordinates
type NetworkGeocoder interface {
	GeocodeNetwork(ipAddress string) (geo.Point, error)
}

// TimezoneGeocoder fills in the timezone of an address that has already been located
type TimezoneGeocoder interface {
	GeocodeTimezone(*geo.Address) error
}

// AddressAutocompleter suggests complete addresses for a partial query
type AddressAutocompleter interface {
	AutocompleteAddress(query string, bias geo.Point) (sliceof.Object[geo.Address], error)
}
