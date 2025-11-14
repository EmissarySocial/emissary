package geocoder

import (
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/sliceof"
)

// Nil is an empty geocoder that returns no results
type Nil struct{}

func NewNil() Nil {
	return Nil{}
}

func (n Nil) GeocodeAddress(address string) (geo.Address, error) {
	return geo.Address{
		Formatted: address,
	}, nil
}

func (n Nil) GeocodeNetwork(ipAddress string) (geo.Point, error) {
	return geo.Point{}, nil
}

func (n Nil) AutocompleteAddress(address string) (sliceof.Object[geo.Address], error) {
	return sliceof.NewObject[geo.Address](), nil
}
