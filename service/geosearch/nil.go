package geosearch

import (
	"github.com/benpate/rosetta/sliceof"
)

// Nil is an empty geosearch adapter that always returns an empty result set
func Nil() GeosearchFunc {
	return func(query string) (sliceof.Object[Place], error) {
		return sliceof.NewObject[Place](), nil
	}
}
