package service

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/outbox"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchQuery defines a service that manages all searchable tags in a domain.
type SearchQuery struct {
	collection       data.Collection
	domainService    *Domain
	followerService  *Follower
	ruleService      *Rule
	searchTagService *SearchTag
	activityStream   *ActivityStream
	host             string
}

// NewSearchQuery returns a fully initialized SearchQuery service
func NewSearchQuery() SearchQuery {
	return SearchQuery{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchQuery) Refresh(collection data.Collection, domainService *Domain, followerService *Follower, ruleService *Rule, searchTagService *SearchTag, activityStream *ActivityStream, host string) {
	service.collection = collection
	service.domainService = domainService
	service.followerService = followerService
	service.ruleService = ruleService
	service.searchTagService = searchTagService
	service.activityStream = activityStream
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *SearchQuery) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *SearchQuery) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe SearchQuerys that match the provided criteria
func (service *SearchQuery) Query(criteria exp.Expression, options ...option.Option) ([]model.SearchQuery, error) {
	result := make([]model.SearchQuery, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the SearchQuerys that match the provided criteria
func (service *SearchQuery) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an SearchQuery from the database
func (service *SearchQuery) Load(criteria exp.Expression, searchQuery *model.SearchQuery) error {

	if err := service.collection.Load(notDeleted(criteria), searchQuery); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Load", "Error loading SearchQuery", criteria)
	}

	return nil
}

// Save adds/updates an SearchQuery in the database
func (service *SearchQuery) Save(searchQuery *model.SearchQuery, note string) error {

	const location = "service.SearchQuery.Save"
	return derp.NewInternalError(location, "Not implemented")
	/*

		if len(searchQuery.Original) > 128 {
			return derp.New(derp.CodeBadRequestError, location, "SearchQuery.Original is too long", searchQuery)
		}

		// Split the query, and Normalize tags and remainder
		if err := service.parseHashtags(searchQuery); err != nil {
			return derp.Wrap(err, location, "Error normalizing tags", searchQuery)
		}

		// RULE: Do not allow global searches here.
		if searchQuery.IsEmpty() {
			return derp.New(derp.CodeBadRequestError, location, "SearchQuery is empty", searchQuery)
		}

		// Save the searchQuery to the database
		if err := service.collection.Save(searchQuery, note); err != nil {
			return derp.Wrap(err, "service.SearchQuery.Save", "Error saving SearchQuery", searchQuery, note)
		}

		// TODO: Add a queue task to try to delete this SearchQuery if it hasn't been subscribed after 1 day

		return nil
	*/
}

func (service *SearchQuery) Upsert(searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.Upsert"

	return derp.NewInternalError(location, "Not implemented")

	/*
		// If the SearchQuery already has an ID then just save it.
		if !searchQuery.SearchQueryID.IsZero() {

			if err := service.Save(searchQuery, "Upsert"); err != nil {
				return derp.Wrap(err, location, "Error saving SearchQuery", searchQuery)
			}

			return nil
		}

		// Fall through means we're searching first, then saving/creating

		// Normalize the Hashtags and Remainder before searching
		if err := service.parseHashtags(searchQuery); err != nil {
			return derp.Wrap(err, location, "Error validating query string", searchQuery.Original)
		}

		// If the SearchQuery is empty, then there's nothing to create. Return an error.
		if searchQuery.IsEmpty() {
			return derp.New(derp.CodeBadRequestError, location, "SearchQuery is empty", searchQuery)
		}

		// Try to find other SearchTags that match this query
		err := service.LoadByTagsAndRemainder(searchQuery.Tags, searchQuery.Remainder, searchQuery)

		// Success..
		if err == nil {
			return nil
		}

		// "Not Found" means that we should just save this as a new record
		if derp.NotFound(err) {

			if err := service.Save(searchQuery, "Upsert"); err != nil {
				return derp.Wrap(err, location, "Error saving SearchQuery", searchQuery)
			}

			return nil
		}

		// Otherwise, you have failed.  Report to the Principal's office.
		return derp.Wrap(err, location, "Error loading SearchQuery", searchQuery)
	*/
}

// Delete removes an SearchQuery from the database (virtual delete)
func (service *SearchQuery) Delete(searchQuery *model.SearchQuery, note string) error {

	// Delete this SearchQuery
	if err := service.collection.Delete(searchQuery, note); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Delete", "Error deleting SearchQuery", searchQuery, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *SearchQuery) LoadOrCreate(queryValues url.Values) (model.SearchQuery, error) {

	const location = "service.SearchQuery.LoadByQueryString"
	return model.SearchQuery{}, derp.NewInternalError(location, "Not implemented")
	/*
		result := model.NewSearchQuery()

		// If we have a searchID token, then try to use it first.
		if token := queryValues.Get("id"); token != "" {
			if err := service.LoadByToken(token, &result); err != nil {
				return result, derp.Wrap(err, location, "Error loading SearchQuery by token", token)
			}
		}

		// Collect the query values into a new SearchQuery object
		result.Parse(queryValues)

		// Fall through means there's no token, or a deleted token.
		if query := queryValues.Get("q"); query != "" {

			result.Query = query

			if err := service.parseHashtags(&result); err != nil {
				return result, derp.Wrap(err, location, "Error normalizing tags", result.Query)
			}

			if err := service.LoadByTagsAndRemainder(result.TagValues, result.Remainder, &result); err == nil {
				return result, nil
			}

			if err := service.Upsert(&result); err != nil {
				return result, derp.Wrap(err, location, "Error upserting SearchQuery", query)
			}

			return result, nil
		}

		return result, derp.NewBadRequestError(location, "No search query provided", queryValues)
	*/
}

func (service *SearchQuery) LoadByToken(token string, searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.LoadByToken"

	// Parse the token as an ID
	searchQueryID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Error converting token to ObjectID", token)
	}

	// Query the database
	criteria := exp.Equal("_id", searchQueryID)
	return service.Load(criteria, searchQuery)
}

func (service *SearchQuery) LoadByTagsAndRemainder(tags []string, remainder string, searchQuery *model.SearchQuery) error {
	criteria := exp.InAll("tagValues", tags).And(exp.Equal("remainder", remainder))
	return service.Load(criteria, searchQuery)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *SearchQuery) MakeToken(queryString url.Values) (string, error) {

	return "", derp.NewInternalError("service.SearchQuery.MakeToken", "Not implemented")
	/*
		targetSearchQuery, ok := service.parseQueryString(queryString)

		criteria := exp.All()

		if q := queryString.Get("q"); q != "" {
			criteria = criteria.And(exp.Equal("query", q))
		}
	*/
}

func (service *SearchQuery) parseQueryString(queryString url.Values) (model.SearchQuery, bool) {

	result := model.NewSearchQuery()
	notEmpty := false

	if q := queryString.Get("q"); q != "" {
		result.Query = q
		notEmpty = true
	}

	if tags := queryString["tags"]; len(tags) > 0 {
		result.Tags = tags
		notEmpty = true
	}

	// if startDate := queryString.Get("startDate"); startDate != "" {
	//	result.StartDate = startDate
	// }

	// if location := queryString.Get("location"); location != "" {
	//	result.Location = location
	// }

	return result, notEmpty
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided Stream.
func (service *SearchQuery) ActivityPubActor(searchQuery *model.SearchQuery, withFollowers bool) (outbox.Actor, error) {

	const location = "service.SearchQuery.ActivityPubActor"

	// Retrieve the domain and Public Key
	privateKey, err := service.domainService.PrivateKey()

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error getting private key")
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(service.ActivityPubURL(searchQuery), privateKey, outbox.WithClient(service.activityStream)) // TODO: Restore Queue:: , outbox.WithQueue(service.queue))

	// Populate the Actor's ActivityPub Followers, if requested
	if withFollowers {

		// Get a channel of all Followers
		followers, err := service.followerService.ActivityPubFollowersChannel(model.FollowerTypeSearch, searchQuery.SearchQueryID)

		if err != nil {
			return outbox.Actor{}, derp.Wrap(err, location, "Error retrieving followers")
		}

		// Get a filter to prevent sending to "Blocked" followers
		ruleFilter := service.ruleService.Filter(primitive.NilObjectID, WithBlocksOnly())
		followerIDs := ruleFilter.ChannelSend(followers)

		// Add the channel of follower IDs to the Actor
		actor.With(outbox.WithFollowers(followerIDs))
	}

	return actor, nil
}

func (service *SearchQuery) ActivityPubURL(searchQuery *model.SearchQuery) string {
	return service.host + "/.search/" + searchQuery.SearchQueryID.Hex()
}

func (service *SearchQuery) ActivityPubName(searchQuery *model.SearchQuery) string {
	domain := service.domainService.Get()
	return searchQuery.Query + " on " + domain.Label
}

func (service *SearchQuery) ActivityPubFollowersURL(searchQuery *model.SearchQuery) string {
	return service.ActivityPubURL(searchQuery) + "/followers"
}

func (service *SearchQuery) ActivityPubFollowingURL(searchQuery *model.SearchQuery) string {
	return service.ActivityPubURL(searchQuery) + "/following"
}

func (service *SearchQuery) ActivityPubInboxURL(searchQuery *model.SearchQuery) string {
	return service.ActivityPubURL(searchQuery) + "/inbox"
}

func (service *SearchQuery) ActivityPubOutboxURL(searchQuery *model.SearchQuery) string {
	return service.ActivityPubURL(searchQuery) + "/outbox"
}

func (service *SearchQuery) ActivityPubSharesURL(searchQuery *model.SearchQuery) string {
	return service.ActivityPubURL(searchQuery) + "/shares"
}
