package geosearch

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

func Nominatim(searchURL string, userAgent string, referer string) GeosearchFunc {

	const location = "service.geosearch.Nominatim"

	if searchURL == "" {
		searchURL = "https://nominatim.openstreetmap.org"
	}

	return func(query string) (sliceof.Object[Place], error) {

		response := make(sliceof.Object[mapof.Any], 0)

		txn := remote.Get(searchURL+"/search").
			UserAgent(userAgent).
			Header("Referer", referer).
			Query("q", query).
			Query("format", "jsonv2").
			Result(&response)

		if err := txn.Send(); err != nil {
			return nil, derp.Wrap(err, location, "Unable to retrieve search results")
		}

		result := make(sliceof.Object[Place], 0, len(response))

		for _, place := range response {

			result = append(result, Place{
				Name:      place.GetString("display_name"),
				Latitude:  place.GetFloat("lat"),
				Longitude: place.GetFloat("lon"),
			})
		}

		return result, nil
	}
}
