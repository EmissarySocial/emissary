package mastodon

import (
	"net/url"
	"strconv"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot/txn"
)

type tooter[Result any] interface {
	Toot() Result
}

func sliceOfToots[In tooter[Out], Out any](slice []In) []Out {

	results := make([]Out, len(slice))

	for i, value := range slice {
		results[i] = value.Toot()
	}

	return results
}

// queryExpression converts data from a txn.QueryPager into an exp.Expression
// that can be used to filter database queries.
func queryExpression(queryPager txn.QueryPager) exp.Expression {

	result := exp.All()

	params := queryPager.QueryPage()

	if params.MaxID != "" {
		if maxID, err := strconv.ParseInt(params.MaxID, 10, 64); err == nil {
			result = result.AndLessThan("createDate", maxID)
		}
	}

	if params.MinID != "" {
		if minID, err := strconv.ParseInt(params.MinID, 10, 64); err == nil {
			result = result.AndLessThan("createDate", minID)
		}
	}

	if params.SinceID != "" {
		if sinceID, err := strconv.ParseInt(params.SinceID, 10, 64); err == nil {
			result = result.AndLessThan("createDate", sinceID)
		}
	}

	return result
}

// getStreamFromURL is a convenience function that combines the following
// steps: 1) locate the domain from the provided Stream URL, 2) load the
// requested stream from the database, and 3) return the Stream and corresponding
// StreamService to the caller.
func getStreamFromURL(serverFactory *server.Factory, streamURL string) (model.Stream, *service.Stream, error) {

	const location = "handler.getStreamFromURI"

	// Parse the URL to 1) validate it's legit, and 2) extract the domain name
	parsedURL, err := url.Parse(streamURL)

	if err != nil {
		return model.Stream{}, nil, derp.Wrap(err, location, "Invalid URI")
	}

	// Get the factory for this Domain
	factory, err := serverFactory.ByDomainName(parsedURL.Host)

	if err != nil {
		return model.Stream{}, nil, derp.Wrap(err, location, "Unrecognized Domain")
	}

	// Try to load the requested Stream using its URL
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByURL(streamURL, &stream); err != nil {
		return model.Stream{}, nil, derp.Wrap(err, location, "Error loading stream")
	}

	// Return values to the caller.
	return stream, streamService, nil

}
