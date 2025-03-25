package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchDomain defines a service that manages the global domain search actor.
type SearchDomain struct {
	collection       data.Collection
	domainService    *Domain
	followerService  *Follower
	ruleService      *Rule
	searchTagService *SearchTag
	activityStream   *ActivityStream
	queue            *queue.Queue
	host             string
}

// NewSearchDomain returns a fully initialized SearchDomain service
func NewSearchDomain() SearchDomain {
	return SearchDomain{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchDomain) Refresh(collection data.Collection, domainService *Domain, followerService *Follower, ruleService *Rule, searchTagService *SearchTag, activityStream *ActivityStream, queue *queue.Queue, host string) {
	service.collection = collection
	service.domainService = domainService
	service.followerService = followerService
	service.ruleService = ruleService
	service.searchTagService = searchTagService
	service.activityStream = activityStream
	service.queue = queue
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *SearchDomain) Close() {
	// Nothin to do here.
}

/******************************************
 * ActivityPub Methods
 ******************************************/

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided Stream.
func (service *SearchDomain) ActivityPubActor(withFollowers bool) (outbox.Actor, error) {

	const location = "service.SearchDomain.ActivityPubActor"

	// Retrieve the domain and Public Key
	privateKey, err := service.domainService.PrivateKey()

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error getting private key")
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(service.ActivityPubURL(), privateKey, outbox.WithClient(service.activityStream))

	// Populate the Actor's ActivityPub Followers, if requested
	if withFollowers {

		// Get a channel of all Followers
		followers, err := service.followerService.ActivityPubFollowersChannel(model.FollowerTypeSearchDomain, primitive.NilObjectID)

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

func (service *SearchDomain) ActivityPubUsername() string {
	return "search"
}

func (service *SearchDomain) ActivityPubURL() string {
	return service.host + "/@search"
}

func (service *SearchDomain) ActivityPubProfileURL() string {
	return service.host + "/@search"
}

func (service *SearchDomain) ActivityPubName() string {
	domain := service.domainService.Get()
	return "All Search Results on " + domain.Label
}

func (service *SearchDomain) ActivityPubFollowersURL() string {
	return service.ActivityPubURL() + "/pub/followers"
}

func (service *SearchDomain) ActivityPubFollowingURL() string {
	return service.ActivityPubURL() + "/pub/following"
}

func (service *SearchDomain) ActivityPubInboxURL() string {
	return service.ActivityPubURL() + "/pub/inbox"
}

func (service *SearchDomain) ActivityPubOutboxURL() string {
	return service.ActivityPubURL() + "/pub/outbox"
}

func (service *SearchDomain) ActivityPubSharesURL() string {
	return service.ActivityPubURL() + "/pub/shares"
}

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *SearchDomain) WebFinger() digit.Resource {

	usernameWithHost := "search@" + service.Hostname()

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:"+usernameWithHost).
		Alias(service.ActivityPubURL()).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, service.ActivityPubURL())
		// .Link(digit.RelationTypeProfile, model.MimeTypeHTML, service.ActivityPubProfileURL())

	return result
}

func (service *SearchDomain) Hostname() string {
	return domain.NameOnly(service.host)
}
