package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	dt "github.com/benpate/domain"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchDomain defines a service that manages the global domain search actor.
type SearchDomain struct {
	factory          *Factory
	domainService    *Domain
	followerService  *Follower
	ruleService      *Rule
	searchTagService *SearchTag
	queue            *queue.Queue
	host             string
}

// NewSearchDomain returns a fully initialized SearchDomain service
func NewSearchDomain(factory *Factory) SearchDomain {
	return SearchDomain{
		factory: factory,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchDomain) Refresh(domainService *Domain, followerService *Follower, ruleService *Rule, searchTagService *SearchTag, queue *queue.Queue, host string) {
	service.domainService = domainService
	service.followerService = followerService
	service.ruleService = ruleService
	service.searchTagService = searchTagService
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

func (service *SearchDomain) GetJSONLD(session data.Session) (mapof.Any, error) {

	const location = "service.SearchDomain.GetJSONLD"

	// Retrieve the domain and Public Key
	publicKeyPEM, err := service.domainService.PublicKeyPEM(session)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to load public key PEM")
	}

	// Return the result as a JSON-LD document
	actorID := service.ActivityPubURL()
	domain := service.domainService.Get()
	result := map[string]any{
		vocab.AtContext:                 []any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
		vocab.PropertyType:              vocab.ActorTypeService,
		vocab.PropertyID:                service.ActivityPubURL(),
		vocab.PropertyURL:               service.ActivityPubProfileURL(),
		vocab.PropertyPreferredUsername: service.ActivityPubUsername(),
		vocab.PropertyName:              service.ActivityPubName(),
		vocab.PropertyIcon:              domain.IconURL(),
		vocab.PropertyImage:             domain.ImageURL(),
		vocab.PropertyInbox:             service.ActivityPubInboxURL(),
		vocab.PropertyOutbox:            service.ActivityPubOutboxURL(),
		vocab.PropertyFollowers:         service.ActivityPubFollowersURL(),
		vocab.PropertyFollowing:         service.ActivityPubFollowingURL(),
		vocab.PropertyTootDiscoverable:  false,
		vocab.PropertyTootIndexable:     false,

		vocab.PropertyPublicKey: map[string]any{
			vocab.PropertyID:           actorID + "#main-key",
			vocab.PropertyOwner:        actorID,
			vocab.PropertyPublicKeyPEM: publicKeyPEM,
		},
	}

	return result, nil
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided Stream.
func (service *SearchDomain) ActivityPubActor(session data.Session) (outbox.Actor, error) {

	const location = "service.SearchDomain.ActivityPubActor"

	// Retrieve the domain and Public Key
	privateKey, err := service.domainService.PrivateKey(session)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error getting private key")
	}

	activityService := service.factory.ActivityStream(model.ActorTypeSearchDomain, primitive.NilObjectID)

	// Return the ActivityPub Actor
	actor := outbox.NewActor(
		service.ActivityPubURL(),
		privateKey,
		outbox.WithFollowers(service.rangeActivityPubFollowers(session)),
		outbox.WithClient(activityService.Client()),
	)

	return actor, nil
}

func (service *SearchDomain) rangeActivityPubFollowers(session data.Session) iter.Seq[string] {

	return func(yield func(string) bool) {

		// Get a channel of all Followers
		followers := service.followerService.RangeActivityPubByType(session, model.FollowerTypeSearchDomain, primitive.NilObjectID)

		for follower := range followers {
			if !yield(follower.Actor.ProfileURL) {
				return // Stop iterating if the yield function returns false
			}
		}
	}
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
	return dt.NameOnly(service.host)
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *SearchDomain) RangeActivityPubFollowers(session data.Session) iter.Seq[string] {
	followers := service.followerService.RangeActivityPubByType(session, model.FollowerTypeSearchDomain, primitive.NilObjectID)
	return iterateFollowerAddresses(followers)
}
