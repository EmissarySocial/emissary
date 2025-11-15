package geocoder

import (
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
)

// Nil is an empty geocoder that returns no results
type Nil struct{}

func NewNil() Nil {
	return Nil{}
}

func (n Nil) GeocodeAddress(address string) (geo.Address, error) {
	log.Debug().Msg("NIL Geocoder: GeocodeAddress")
	return geo.Address{
		Formatted: address,
	}, nil
}

func (n Nil) GeocodeNetwork(ipAddress string) (geo.Point, error) {
	log.Debug().Msg("NIL Geocoder: GeocodeNetwork")
	return geo.Point{}, nil
}

func (n Nil) AutocompleteAddress(address string, bias geo.Point) (sliceof.Object[geo.Address], error) {
	log.Debug().Msg("NIL Geocoder: Autocomplete Address")
	return sliceof.NewObject[geo.Address](), nil
}

func (n Nil) GeocodeTimezone(address *geo.Address) error {
	log.Debug().Msg("NIL Geocoder: GeocodeTimezone")
	return nil
}
