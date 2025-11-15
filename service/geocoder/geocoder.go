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

type TimezoneGeocoder interface {
	GeocodeTimezone(*geo.Address) error
}

type AddressAutocompleter interface {
	AutocompleteAddress(query string, bias geo.Point) (sliceof.Object[geo.Address], error)
}
