package service

import (
	"iter"
	"net/url"
	"slices"
	"strings"

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

// Range returns an iterator that contains all of the SearchQuerys that match the provided criteria
func (service *SearchQuery) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.SearchQuery], error) {
	it, err := service.collection.Iterator(notDeleted(criteria), options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.SearchQuery.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(it, model.NewSearchQuery), nil
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

	if len(searchQuery.Query) > 128 {
		return derp.New(derp.CodeBadRequestError, location, "SearchQuery.Original is too long", searchQuery)
	}

	// RULE: Do not allow global searches here.
	if searchQuery.IsEmpty() {
		return derp.New(derp.CodeBadRequestError, location, "SearchQuery is empty", searchQuery)
	}

	// Save the searchQuery to the database
	if err := service.collection.Save(searchQuery, note); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Save", "Error saving SearchQuery", searchQuery, note)
	}

	// TODO: LOW: Add a queue task to try to delete this SearchQuery if it hasn't been subscribed after 1 day

	return nil
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

// RangeAll returns an iterator that contains all of the SearchQuerys in the database
func (service *SearchQuery) RangeAll() (iter.Seq[model.SearchQuery], error) {
	return service.Range(exp.All())
}

// LoadByID retrieves a SearchQuery using the provided ID
func (service *SearchQuery) LoadByID(searchQueryID primitive.ObjectID, searchQuery *model.SearchQuery) error {
	criteria := exp.Equal("_id", searchQueryID)
	return service.Load(criteria, searchQuery)
}

// LoadByToken retrieves a SearchQuery using the provided token
func (service *SearchQuery) LoadByToken(token string, searchQuery *model.SearchQuery) error {

	// Parse the token as an ID
	searchQueryID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.SearchQuery.LoadByToken", "Error converting token to ObjectID", token)
	}

	return service.LoadByID(searchQueryID, searchQuery)
}

// LoadOrCreate creates/retrieves a SearchQuery using the provided queryValues
func (service *SearchQuery) LoadOrCreate(queryValues url.Values) (model.SearchQuery, error) {

	const location = "service.SearchQuery.LoadOrCreate"

	// Parse the query values into a temporary SearchQuery
	newSearchQuery, isPopulated := service.parseQueryValues(queryValues)

	if !isPopulated {
		return model.NewSearchQuery(), derp.NewBadRequestError(location, "No useful data in queryValues", queryValues)
	}

	// Build search criteria to see if this SearchQuery already exists
	criteria := exp.
		Equal("types", newSearchQuery.Types).
		AndEqual("tags", newSearchQuery.Tags).
		AndEqual("index", newSearchQuery.Index)

	// Try to find the SearchQuery in the database
	existingSearchQuery := model.NewSearchQuery()
	err := service.Load(criteria, &existingSearchQuery)

	// If it already exists, then return the ID
	if err == nil {
		return existingSearchQuery, nil
	}

	// If it doesn't exist, then create a new record and return it
	if derp.NotFound(err) {

		if err := service.Save(&newSearchQuery, "LoadOrCreate"); err != nil {
			return model.NewSearchQuery(), derp.Wrap(err, location, "Error saving SearchQuery", newSearchQuery)
		}

		return newSearchQuery, nil
	}

	// Fall through to a real error querying the database
	return model.NewSearchQuery(), derp.Wrap(err, location, "Error searching for SearchQuery", criteria)
}

func (service *SearchQuery) parseQueryValues(queryValues url.Values) (model.SearchQuery, bool) {

	result := model.NewSearchQuery()
	isPopulated := false

	// Parse "types" into the SearchQuery
	if types := queryValues.Get("types"); types != "" {
		result.Types = strings.Split(types, ",")
	}

	// Parse the query into the SearchQuery
	if q := queryValues.Get("q"); q != "" {
		result.SetQuery(q)
	}

	// Parse "tags" into the SearchQuery
	if tags := queryValues["tags"]; len(tags) > 0 {
		result.AppendTags(tags...)
	}

	// Sort all slices so they can be compared correctly

	if len(result.Types) > 0 {
		slices.Sort(result.Types)
		isPopulated = true
	}

	if len(result.Index) > 0 {
		slices.Sort(result.Index)
		isPopulated = true
	}

	if len(result.Tags) > 0 {
		slices.Sort(result.Tags)
		isPopulated = true
	}

	// if startDate := queryString.Get("startDate"); startDate != "" {
	//	result.StartDate = startDate
	// }

	// if location := queryString.Get("location"); location != "" {
	//	result.Location = location
	// }

	return result, isPopulated
}

/******************************************
 * ActivityPub Methods
 ******************************************/

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
