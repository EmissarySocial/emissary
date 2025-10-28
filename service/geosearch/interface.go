package geosearch

import (
	"github.com/benpate/rosetta/sliceof"
)

type Place struct {
	Name      string
	Latitude  float64
	Longitude float64
}

type GeosearchFunc func(string) (sliceof.Object[Place], error)
