package service

import (
	"iter"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	dt "github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchQuery defines a service that manages all searchable tags in a domain.
type SearchQuery struct {
	factory          *Factory
	domainService    *Domain
	followerService  *Follower
	ruleService      *Rule
	searchTagService *SearchTag
	queue            *queue.Queue
	host             string
}

// NewSearchQuery returns a fully initialized SearchQuery service
func NewSearchQuery(factory *Factory) SearchQuery {
	return SearchQuery{
		factory: factory,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchQuery) Refresh(domainService *Domain, followerService *Follower, ruleService *Rule, searchTagService *SearchTag, queue *queue.Queue, host string) {
	service.domainService = domainService
	service.followerService = followerService
	service.ruleService = ruleService
	service.searchTagService = searchTagService
	service.queue = queue
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *SearchQuery) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *SearchQuery) collection(session data.Session) data.Collection {
	return session.Collection("SearchQuery")
}

func (service *SearchQuery) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice of allthe SearchQuerys that match the provided criteria
func (service *SearchQuery) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.SearchQuery, error) {
	result := make([]model.SearchQuery, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// Range returns an iterator that contains all of the SearchQuerys that match the provided criteria
func (service *SearchQuery) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.SearchQuery], error) {
	it, err := service.collection(session).Iterator(notDeleted(criteria), options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.SearchQuery.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(it, model.NewSearchQuery), nil
}

// LoadWithSoftDeletes searches ALL SearchQuery in the database, including those that have been "soft deleted"
func (service *SearchQuery) LoadWithSoftDeletes(session data.Session, criteria exp.Expression, searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.LoadWithSoftDeletes"

	if err := service.collection(session).Load(criteria, searchQuery); err != nil {
		return derp.Wrap(err, location, "Unable to load SearchQuery", criteria)
	}

	// If the SearchQuery has been deleted, then restore it before returning
	if searchQuery.IsDeleted() {

		searchQuery.DeleteDate = 0

		if err := service.Save(session, searchQuery, "Restored"); err != nil {
			return derp.Wrap(err, location, "Error restoring SearchQuery", searchQuery)
		}
	}

	return nil
}

// Save adds/updates an SearchQuery in the database
func (service *SearchQuery) Save(session data.Session, searchQuery *model.SearchQuery, note string) error {

	const location = "service.SearchQuery.Save"

	if len(searchQuery.Query) > 128 {
		return derp.BadRequestError(location, "SearchQuery.Original is too long", searchQuery)
	}

	// RULE: Do not allow global searches here.
	if searchQuery.IsEmpty() {
		return derp.BadRequestError(location, "SearchQuery is empty", searchQuery)
	}

	// Normalize all slices and make query signature
	searchQuery.MakeSignature()
	wasNew := searchQuery.IsNew()

	// Save the searchQuery to the database
	if err := service.collection(session).Save(searchQuery, note); err != nil {
		return derp.Wrap(err, location, "Unable to save SearchQuery", searchQuery, note)
	}

	// Add a queue task to delete this SearchQuery if it has no followers after 12 hour.
	if wasNew {
		service.queue.NewTask(
			"DeleteEmptySearchQuery",
			mapof.Any{
				"host":          service.host,
				"searchQueryID": searchQuery.SearchQueryID.Hex(),
			},
			queue.WithDelayHours(1),
		)
	}

	return nil
}

// Delete removes an SearchQuery from the database (virtual delete)
func (service *SearchQuery) Delete(session data.Session, searchQuery *model.SearchQuery, note string) error {

	// Delete this SearchQuery
	if err := service.collection(session).Delete(searchQuery, note); err != nil {
		return derp.Wrap(err, "service.SearchQuery.Delete", "Error deleting SearchQuery", searchQuery, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

// RangeAll returns an iterator that contains all of the SearchQuerys in the database
func (service *SearchQuery) RangeAll(session data.Session) (iter.Seq[model.SearchQuery], error) {
	return service.Range(session, exp.All())
}

// RangeNearMatches searces for SearchQueries that *might* match the provided SearchResult. It queries
// easily matchable values, but it is not thorough, so it WILL return false positives. This at least
// partially limits the number of SearchQueries that need to be scanned by the application server.
func (service *SearchQuery) RangeNearMatches(session data.Session, searchResult *model.SearchResult) (iter.Seq[model.SearchQuery], error) {

	var criteria exp.Expression = exp.And()

	// searchQuery.Type is "" or matches searchResult
	criteria = criteria.And(
		exp.Or(
			exp.Equal("type", ""),
			exp.Equal("type", searchResult.Type),
		),
	)

	return service.Range(session, criteria)
}

// LoadByID retrieves a SearchQuery using the provided ID
func (service *SearchQuery) LoadByID(session data.Session, searchQueryID primitive.ObjectID, searchQuery *model.SearchQuery) error {

	const location = "service.SearchQuery.LoadByID"

	criteria := exp.Equal("_id", searchQueryID)

	if err := service.LoadWithSoftDeletes(session, criteria, searchQuery); err != nil {
		return derp.Wrap(err, location, "Unable to load SearchQuery", searchQueryID)
	}

	return nil
}

// LoadByToken retrieves a SearchQuery using the provided token
func (service *SearchQuery) LoadByToken(session data.Session, token string, searchQuery *model.SearchQuery) error {

	// Parse the token as an ID
	searchQueryID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.SearchQuery.LoadByToken", "Error converting token to ObjectID", token)
	}

	return service.LoadByID(session, searchQueryID, searchQuery)
}

// LoadBySignature retrieves a SearchQuery using the provided signature
func (service *SearchQuery) LoadBySignature(session data.Session, signature string, searchQuery *model.SearchQuery) error {
	criteria := exp.Equal("signature", signature)
	return service.LoadWithSoftDeletes(session, criteria, searchQuery)
}

// LoadOrCreate creates/retrieves a SearchQuery using the provided queryValues
func (service *SearchQuery) LoadOrCreate(session data.Session, queryValues url.Values) (model.SearchQuery, error) {

	const location = "service.SearchQuery.LoadOrCreate"

	// Parse the query values into a temporary SearchQuery
	newSearchQuery, isPopulated := service.parseQueryValues(queryValues)

	if !isPopulated {
		return model.NewSearchQuery(), derp.BadRequestError(location, "No useful data in queryValues", queryValues)
	}

	// Try to find the SearchQuery in the database
	existingSearchQuery := model.NewSearchQuery()
	err := service.LoadBySignature(session, newSearchQuery.Signature, &existingSearchQuery)

	// If it already exists, then return the ID
	if err == nil {
		return existingSearchQuery, nil
	}

	// If it doesn't exist, then create a new record and return it
	if derp.IsNotFound(err) {

		if err := service.Save(session, &newSearchQuery, "LoadOrCreate"); err != nil {
			return model.NewSearchQuery(), derp.Wrap(err, location, "Unable to save SearchQuery", newSearchQuery)
		}

		return newSearchQuery, nil
	}

	// Fall through to a real error querying the database
	return model.NewSearchQuery(), derp.Wrap(err, location, "Error locating SearchQuery")
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

	// if startDate := queryString.Get("startDate"); startDate != "" {
	//	result.StartDate = startDate
	// }

	// if location := queryString.Get("location"); location != "" {
	//	result.Location = location
	// }

	// Create the "signature" value for this SearchQuery
	result.MakeSignature()

	// Determine if this has been populated or not
	if len(result.Types) > 0 {
		isPopulated = true
	}

	if len(result.Index) > 0 {
		isPopulated = true
	}

	if len(result.Tags) > 0 {
		isPopulated = true
	}

	return result, isPopulated
}

/******************************************
 * ActivityPub Methods
 ******************************************/

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided Stream.
func (service *SearchQuery) ActivityPubActor(session data.Session, searchQueryID primitive.ObjectID) (outbox.Actor, error) {

	const location = "service.SearchQuery.ActivityPubActor"

	// Retrieve the domain and Public Key
	privateKey, err := service.domainService.PrivateKey(session)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error getting private key")
	}

	activityService := service.factory.ActivityStream(model.ActorTypeSearchQuery, searchQueryID)

	// Return the ActivityPub Actor
	actor := outbox.NewActor(
		service.ActivityPubURL(searchQueryID),
		privateKey,
		outbox.WithFollowers(service.rangeActivityPubFollowers(session, searchQueryID)),
		outbox.WithClient(activityService.Client()),
		// TODO: Restore Queue:: , outbox.WithQueue(service.queue))
	)

	return actor, nil
}

func (service *SearchQuery) rangeActivityPubFollowers(session data.Session, searchQueryID primitive.ObjectID) iter.Seq[string] {

	return func(yield func(string) bool) {

		followers := service.followerService.RangeActivityPubByType(session, model.FollowerTypeSearch, searchQueryID)

		for follower := range followers {
			if !yield(follower.Actor.ProfileURL) {
				return // Stop iterating if the yield function returns false
			}
		}
	}
}

func (service *SearchQuery) ActivityPubUsername(searchQueryID primitive.ObjectID) string {
	return "search_" + searchQueryID.Hex()
}

func (service *SearchQuery) ActivityPubURL(searchQueryID primitive.ObjectID) string {
	return service.host + "/@" + service.ActivityPubUsername(searchQueryID)
}

func (service *SearchQuery) ActivityPubProfileURL(searchQuery *model.SearchQuery) string {
	return searchQuery.URL
}

func (service *SearchQuery) ActivityPubName(searchQuery *model.SearchQuery) string {
	domain := service.domainService.Get()

	if query := searchQuery.Query; query != "" {
		return searchQuery.Query + " on " + domain.Label
	} else {
		return "Search everything " + domain.Label
	}
}

func (service *SearchQuery) ActivityPubFollowersURL(searchQueryID primitive.ObjectID) string {
	return service.ActivityPubURL(searchQueryID) + "/pub/followers"
}

func (service *SearchQuery) ActivityPubFollowingURL(searchQueryID primitive.ObjectID) string {
	return service.ActivityPubURL(searchQueryID) + "/pub/following"
}

func (service *SearchQuery) ActivityPubInboxURL(searchQueryID primitive.ObjectID) string {
	return service.ActivityPubURL(searchQueryID) + "/pub/inbox"
}

func (service *SearchQuery) ActivityPubOutboxURL(searchQueryID primitive.ObjectID) string {
	return service.ActivityPubURL(searchQueryID) + "/pub/outbox"
}

func (service *SearchQuery) ActivityPubSharesURL(searchQueryID primitive.ObjectID) string {
	return service.ActivityPubURL(searchQueryID) + "/pub/shares"
}

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *SearchQuery) WebFinger(session data.Session, token string) (digit.Resource, error) {

	const location = "service.SearchQuery.LoadWebFinger"

	// Load the SearchQuery from the database
	searchQuery := model.NewSearchQuery()

	// If the token begins with a question mark, then it's a query string
	// and we need to parse this into a SearchQuery.
	if token, found := strings.CutPrefix(token, "?"); found {

		queryValues, err := url.ParseQuery(token)

		if err != nil {
			return digit.Resource{}, derp.BadRequestError(location, "Invalid Query String", token)
		}

		searchQuery, err = service.LoadOrCreate(session, queryValues)

		if err != nil {
			return digit.Resource{}, derp.Wrap(err, location, "Unable to load SearchQuery", queryValues)
		}

	} else {
		if err := service.LoadByToken(session, token, &searchQuery); err != nil {
			return digit.Resource{}, derp.BadRequestError(location, "Invalid Token", token)
		}
	}

	username := service.ActivityPubUsername(searchQuery.SearchQueryID)
	usernameWithHost := username + "@" + service.Hostname()

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:"+usernameWithHost).
		Alias(service.ActivityPubURL(searchQuery.SearchQueryID)).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, service.ActivityPubURL(searchQuery.SearchQueryID)).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, service.ActivityPubProfileURL(&searchQuery))

	return result, nil
}

func (service *SearchQuery) Hostname() string {
	return dt.NameOnly(service.host)
}
