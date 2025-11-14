package geocoder

import (
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/sliceof"
)

type AddressGeocoder interface {
	GeocodeAddress(string) (geo.Address, error)
}

type NetworkGeocoder interface {
	GeocodeNetwork(ipAddress string) (geo.Point, error)
}

type TimezomeGeocoder interface {
	GeocodeTimezone(string) (string, error)
}

type AddressAutocompleter interface {
	AutocompleteAddress(address string) (sliceof.Object[geo.Address], error)
}
