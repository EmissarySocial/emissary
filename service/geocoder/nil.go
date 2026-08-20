package geocoder

import (
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
)

// Nil is an empty geocoder that returns no results
type Nil struct{}

// NewNil returns a Nil geocoder
func NewNil() Nil {
	return Nil{}
}

// GeocodeAddress resolves a free-text address into a geo.Address. Implements the AddressGeocoder interface.
func (n Nil) GeocodeAddress(address string) (geo.Address, error) {
	log.Debug().Msg("NIL Geocoder: GeocodeAddress")
	return geo.Address{
		Formatted: address,
	}, nil
}

// GeocodeNetwork resolves an IP address into approximate coordinates. Implements the NetworkGeocoder interface.
func (n Nil) GeocodeNetwork(ipAddress string) (geo.Point, error) {
	log.Debug().Msg("NIL Geocoder: GeocodeNetwork")
	return geo.Point{}, nil
}

// AutocompleteAddress returns address suggestions for a partial query, biased toward the provided point. Implements the AddressAutocompleter interface.
func (n Nil) AutocompleteAddress(address string, bias geo.Point) (sliceof.Object[geo.Address], error) {
	log.Debug().Msg("NIL Geocoder: Autocomplete Address")
	return sliceof.NewObject[geo.Address](), nil
}

// GeocodeTimezone fills in the timezone of the provided address. Implements the TimezoneGeocoder interface.
func (n Nil) GeocodeTimezone(address *geo.Address) error {
	log.Debug().Msg("NIL Geocoder: GeocodeTimezone")
	return nil
}
