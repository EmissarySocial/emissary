package service

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

type TilesGeocoder interface {
	GetTileURL() string
}

type AddressAutocompleter interface {
	AutocompleteAddress(address string) (sliceof.Object[geo.Address], error)
}
