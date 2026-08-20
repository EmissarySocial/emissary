package mastodon

import (
	"net/url"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/txn"
)

// tootGetter is any ranked model object that can render itself as a Mastodon API object
type tootGetter[Result any] interface {
	Toot() Result
	rankGetter
}

// rankGetter is any object that exposes the rank used to paginate Mastodon API results
type rankGetter interface {
	GetRank() int64
}

// getSliceOfToots maps a slice of "tootGetters" into a slice of toot objects
// The specific type of the result objects is determined by the `Toot()` method.
func getSliceOfToots[In tootGetter[Out], Out any](slice []In) []Out {

	results := make([]Out, len(slice))

	for i, value := range slice {
		results[i] = value.Toot()
	}

	return results
}

// getPageInfo uses the GetRank() interface method to calclate
// the MaxID and MinID values for a slice of tootGetters
func getPageInfo[In rankGetter](slice []In) toot.PageInfo {

	result := toot.PageInfo{}

	if length := len(slice); length > 0 {
		last := length - 1
		result.MaxID = strconv.FormatInt(slice[last].GetRank(), 10)
		result.MinID = strconv.FormatInt(slice[0].GetRank(), 10)
	}

	return result
}

// queryExpression converts data from a txn.QueryPager into an exp.Expression
// that can be used to filter database queries.
func queryExpression(queryPager txn.QueryPager) exp.Expression {

	var result exp.Expression = exp.All()

	params := queryPager.QueryPage()

	if params.MinID != "" {
		if minID, err := strconv.ParseInt(params.MinID, 10, 64); err == nil {
			result = result.AndLessThan("createDate", minID)
		}
	}

	if params.MaxID != "" {
		if maxID, err := strconv.ParseInt(params.MaxID, 10, 64); err == nil {
			result = result.AndLessThan("createDate", maxID)
		}
	}

	if params.SinceID != "" {
		if sinceID, err := strconv.ParseInt(params.SinceID, 10, 64); err == nil {
			result = result.AndGreaterThan("createDate", sinceID)
		}
	}

	return result
}

// getStreamFromURL is a convenience function that combines the following
// steps: 1) locate the domain from the provided Stream URL, 2) load the
// requested stream from the database, and 3) return the Stream and corresponding
// StreamService to the caller.
func getStreamFromURL(serverFactory *server.Factory, streamURL string) (*service.Factory, *service.Stream, model.Stream, error) {

	const location = "handler.getStreamFromURI"

	// Parse the URL to 1) validate it's legit, and 2) extract the domain name
	parsedURL, err := url.Parse(streamURL)

	if err != nil {
		return nil, nil, model.Stream{}, derp.Wrap(err, location, "Invalid URI")
	}

	// Get the factory for this Domain
	factory, err := serverFactory.ByHostname(parsedURL.Host)

	if err != nil {
		return nil, nil, model.Stream{}, derp.Wrap(err, location, "Unrecognized Domain")
	}

	// Get a database session for this request
	session, cancel, err := factory.Session(time.Minute)

	if err != nil {
		return nil, nil, model.Stream{}, derp.Wrap(err, location, "Creating session")
	}

	defer cancel()

	// Try to load the requested Stream using its URL
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByURL(session, streamURL, &stream); err != nil {
		return nil, nil, model.Stream{}, derp.Wrap(err, location, "Loading stream")
	}

	// Return values to the caller.
	return factory, streamService, stream, nil

}

// userCanStream loads the Template for the provided Stream and verifies that the
// authorization is allowed to perform the named READ action (e.g. "view") on it.
// It returns nil when the action is permitted, and a Forbidden error (or a wrapped
// internal error) otherwise. Reads follow the Stream's template/state visibility
// policy, so a published post is world-readable while a restricted one is not.
//
// Write operations (edit, delete) do NOT use this gate: the Mastodon API's contract
// is that a status belongs to one account, so writes are author-only via
// userOwnsStream, independent of whatever roles the template happens to grant.
func userCanStream(factory *service.Factory, session data.Session, authorization *model.Authorization, stream *model.Stream, actionID string) error {

	const location = "handler.mastodon.userCanStream"

	// Load the Template for this Stream
	template, err := factory.Template().Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Loading template")
	}

	// Check permissions for the requested action
	allowed, err := factory.Permission().UserCan(session, authorization, &template, stream, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Checking permissions")
	}

	if !allowed {
		return derp.Forbidden(location, "User is not authorized to perform this action", "ActionID: "+actionID)
	}

	return nil
}

// userOwnsStream verifies that the authorization is allowed to modify (edit or
// delete) the provided Stream via the Mastodon API. This mirrors Mastodon's model,
// where a status belongs to a single account: only the author may modify it. Domain
// owners are also permitted, so server moderation can take a post down. Unlike
// userCanStream, this deliberately ignores the template's access list — a Mastodon
// write must never be granted to a non-author just because a template shares an
// "edit" or "delete" role with a group.
func userOwnsStream(authorization *model.Authorization, stream *model.Stream) error {

	const location = "handler.mastodon.userOwnsStream"

	if stream.IsAuthor(authorization.UserID) {
		return nil
	}

	if authorization.DomainOwner {
		return nil
	}

	return derp.Forbidden(location, "User is not authorized to modify this stream")
}
