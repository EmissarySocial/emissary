package service

/******************************************
 * Outbox2 Delivery
 *
 * Emissary's side of hannibal's delivery pipeline: who signs for a
 * local actor, which collections are that actor's own, who is
 * excluded, and where email goes. Hannibal delivers to whoever an
 * activity names, so every decision about addressing is made here
 * or by the writer that built the activity.
 ******************************************/

import (
	"context"
	"iter"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/floodgate/remotegate"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskOutbox2Email is the queue task that emails one activity to one recipient
const TaskOutbox2Email = "Outbox2-Email"

// floodgateTimeout bounds the query that a fan-out makes for Floodgate's blocked hosts
const floodgateTimeout = 30 * time.Second

// Pipeline returns the delivery pipeline for activities sent by one local actor. It is built
// for a single task, because its lookups run on that task's session.
func (service *Outbox2) Pipeline(session data.Session, actorURL string, allowPrivateIPs bool) delivery.Pipeline {

	actorType, actorID := service.localActor(actorURL)
	rulesUserID := ruleUserID(actorType, actorID)
	sendLocator := service.getSendLocator(session)

	collections := deliveryCollections{
		session:          session,
		host:             service.host,
		locator:          service.locator,
		followerService:  service.followerService,
		circleService:    service.circleService,
		privilegeService: service.privilegeService,
		identityService:  service.identityService,
		actorType:        actorType,
		actorID:          actorID,
	}

	return delivery.New(
		service.queue,
		service.activityService.AppClient(),
		func(actorID string) (delivery.Signer, error) {
			return sendLocator.Actor(actorID)
		},
		delivery.WithBlockedHosts(service.blockedHosts),
		delivery.WithRemoteOptions(service.remoteOptions),
		delivery.WithCollections(collections),
		delivery.WithFilter(func(string) (func(string) bool, error) {
			return service.deliveryFilter(session, rulesUserID)
		}),
		delivery.WithFilterOne(func(_ string, recipientURL string) bool {
			return service.deliveryAllowed(session, rulesUserID, recipientURL)
		}),
		delivery.WithEmail(service.emailPublisher(actorURL)),
		delivery.WithAllowPrivateIPs(allowPrivateIPs),
	)
}

// blockedHosts returns the hosts that Floodgate has marked dormant or paused, which a fan-out
// queues nothing for
func (service *Outbox2) blockedHosts() (delivery.HostList, error) {

	ctx, cancel := context.WithTimeout(context.Background(), floodgateTimeout)
	defer cancel()

	blocked, err := service.floodgate().Blocked(ctx)

	if err != nil {
		return nil, derp.Wrap(err, "service.Outbox2.blockedHosts", "Loading blocked hosts")
	}

	return blocked, nil
}

// remoteOptions returns the options added to one delivery's request. Floodgate's option asks
// before the request is sent and records its outcome, so nothing else reports to Floodgate.
func (service *Outbox2) remoteOptions() []remote.Option {

	// RULE: Build a new option for every request, because it carries state between its hooks
	return []remote.Option{remotegate.Option(service.floodgate())}
}

// localActor returns the type and ID of the local User or Stream that an actor URL names, and
// an empty type for any other URL
func (service *Outbox2) localActor(actorURL string) (string, primitive.ObjectID) {

	// The followers URI parser already tells a User URL from a Stream URL, so reuse its rules
	return parseFollowersURI(service.host, actorURL+"/pub/followers")
}

// deliveryFilter returns the test that every recipient of one fan-out must pass: the sender's
// block rules, loaded once, and the network boundary
func (service *Outbox2) deliveryFilter(session data.Session, rulesUserID primitive.ObjectID) (func(string) bool, error) {

	const location = "service.Outbox2.deliveryFilter"

	notBlocked, err := service.ruleService.DeliveryFilter(session, rulesUserID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading block rules")
	}

	return func(recipientURL string) bool {

		if service.crossesNetworkBoundary(recipientURL) {
			return false
		}

		return notBlocked(recipientURL)
	}, nil
}

// deliveryAllowed tests one recipient against the same rules as deliveryFilter, by a query of
// its own. A send task uses it for a recipient that the fan-out never saw.
func (service *Outbox2) deliveryAllowed(session data.Session, rulesUserID primitive.ObjectID, recipientURL string) bool {

	if service.crossesNetworkBoundary(recipientURL) {
		return false
	}

	return !service.ruleService.DeliveryBlocked(session, rulesUserID, recipientURL)
}

// crossesNetworkBoundary returns TRUE if a recipient is on the other side of the line between
// local and public networks from this domain
func (service *Outbox2) crossesNetworkBoundary(recipientURL string) bool {

	// RULE: An email address has no host to compare. Whether this server sends email at all is
	// decided by its SMTP configuration.
	if delivery.IsEmail(recipientURL) {
		return false
	}

	// RULE: A local domain sends only to local hosts, and a public domain only to public ones.
	// A development server must never deliver to real followers.
	return uri.IsLocalHostname(recipientURL) != uri.IsLocalHostname(service.host)
}

// emailPublisher returns the function that queues one email task for a mailto: recipient
func (service *Outbox2) emailPublisher(actorURL string) func(string, string, map[string]any) error {

	const location = "service.Outbox2.emailPublisher"

	// RULE: Ask once per fan-out, not once per recipient. A domain that runs without email
	// would otherwise report the same error for every email follower it has.
	canSendEmail := service.domainEmail.IsConfigured()

	if !canSendEmail {
		log.Debug().Str("location", location).Msg("SMTP is not configured. Skipping email recipients.")
	}

	return func(activityID string, address string, activity map[string]any) error {

		if !canSendEmail {
			return nil
		}

		// RULE: The signature keeps a retried fan-out from queueing a second email while the first waits
		task := queue.NewTask(
			TaskOutbox2Email,
			mapof.Any{
				"hostname":   uri.Hostname(service.host),
				"actor":      actorURL,
				"address":    strings.TrimPrefix(address, "mailto:"),
				"activityId": activityID,
				"activity":   activity,
			},
			queue.WithSignature(activityID+" "+address),
		)

		if err := service.queue.Publish(task); err != nil {
			return derp.Wrap(err, location, "Publishing email task")
		}

		return nil
	}
}

// SendEmail emails one activity to one address, on behalf of the local actor that sent it.
// The Outbox2-Email task calls it.
func (service *Outbox2) SendEmail(session data.Session, actorURL string, address string, activity mapof.Any) error {

	const location = "service.Outbox2.SendEmail"

	_, actorID := service.localActor(actorURL)

	// The email is built from the recipient's Follower record, which holds their unsubscribe link
	follower := model.NewFollower()
	err := service.followerService.LoadByEmailAddress(session, actorID, address, &follower)

	// RULE: Someone who is in a circle by email, without following, has no Follower record, and
	// no email is designed for them yet. They are skipped, not failed.
	if derp.IsNotFound(err) {
		log.Debug().Str("location", location).Msg("Email recipient is not a Follower. Skipping.")
		return nil
	}

	if err != nil {
		return derp.Wrap(err, location, "Loading Follower")
	}

	// RULE: Email only confirmed Followers. Anyone can type an address into the follow form,
	// so mailing an unconfirmed one defeats double opt-in (BUG-207 item 2).
	if !follower.IsActive() {
		return nil
	}

	if err := service.domainEmail.SendFollowerActivity(&follower, activity); err != nil {
		return derp.Wrap(err, location, "Sending email")
	}

	return nil
}

// deliveryCollections expands the collections that belong to one local actor: its followers,
// and the circles of a User
type deliveryCollections struct {
	session          data.Session
	host             string
	locator          *Locator
	followerService  *Follower
	circleService    *Circle
	privilegeService *Privilege
	identityService  *Identity
	actorType        string
	actorID          primitive.ObjectID
}

// IsCollection returns TRUE if a URL names a followers collection or a circle on this server,
// whoever owns it. It reads the URL only.
func (collections deliveryCollections) IsCollection(collectionURL string) bool {

	if actorType, _ := parseFollowersURI(collections.host, collectionURL); actorType != "" {
		return true
	}

	_, _, isCircle := collections.locator.ParseCircle(collectionURL)
	return isCircle
}

// Members returns the members of one of the sending actor's own collections. A collection on
// this server that belongs to anyone else yields no one.
func (collections deliveryCollections) Members(collectionURL string) (iter.Seq[string], error) {

	if actorType, actorID := parseFollowersURI(collections.host, collectionURL); actorType != "" {
		return collections.followers(actorType, actorID)
	}

	if userID, circleID, isCircle := collections.locator.ParseCircle(collectionURL); isCircle {
		return collections.circleMembers(userID, circleID)
	}

	return noRecipients, nil
}

// noRecipients yields nothing
func noRecipients(func(string) bool) {}

// followers returns the sending actor's followers: an actor URL for each ActivityPub follower,
// and a mailto: URL for each confirmed email follower
func (collections deliveryCollections) followers(actorType string, actorID primitive.ObjectID) (iter.Seq[string], error) {

	const location = "service.deliveryCollections.followers"

	// RULE: An actor reaches only its own followers. Naming another actor's collection must
	// not deliver to people who never followed the sender.
	if (actorType != collections.actorType) || (actorID != collections.actorID) {
		log.Warn().Str("actorType", actorType).Str("actorId", actorID.Hex()).Msg("service.deliveryCollections: an activity named another actor's followers. No one was added.")
		return noRecipients, nil
	}

	followers, err := collections.followerService.ListFollowers(collections.session, actorType, actorID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading followers")
	}

	return func(yield func(string) bool) {
		for follower := range followers {
			if !yield(followerRecipient(&follower)) {
				return
			}
		}
	}, nil
}

// followerRecipient returns the address that one Follower receives activities at, or an empty
// string for a Follower who receives none
func followerRecipient(follower *model.Follower) string {

	switch follower.Method {

	case model.FollowerMethodActivityPub:
		return follower.Actor.ProfileURL

	case model.FollowerMethodEmail:

		// RULE: Email only confirmed Followers (BUG-207 item 2)
		if !follower.IsActive() {
			return ""
		}

		if follower.Actor.EmailAddress == "" {
			return ""
		}

		return "mailto:" + follower.Actor.EmailAddress

	default:
		return ""
	}
}

// circleMembers returns the members of one of the sending User's circles, whether or not they
// follow: an actor URL for each member who has one, and a mailto: URL for each who has only email
func (collections deliveryCollections) circleMembers(userID primitive.ObjectID, circleID primitive.ObjectID) (iter.Seq[string], error) {

	const location = "service.deliveryCollections.circleMembers"

	// RULE: Only the User who owns a circle can send to it
	if (collections.actorType != model.ActorTypeUser) || (userID != collections.actorID) {
		log.Warn().Str("userId", userID.Hex()).Msg("service.deliveryCollections: an activity named another User's circle. No one was added.")
		return noRecipients, nil
	}

	// RULE: The circle must exist and belong to that User. A deleted circle reaches no one.
	circle := model.NewCircle()
	err := collections.circleService.LoadByID(collections.session, userID, circleID, &circle)

	if derp.IsNotFound(err) {
		return noRecipients, nil
	}

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading circle", "circleId: "+circleID.Hex())
	}

	privileges, err := collections.privilegeService.RangeByCircle(collections.session, circleID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading circle members", "circleId: "+circleID.Hex())
	}

	return func(yield func(string) bool) {
		for privilege := range privileges {
			if !yield(collections.memberRecipient(&privilege)) {
				return
			}
		}
	}, nil
}

// memberRecipient returns the address that one circle member receives activities at
func (collections deliveryCollections) memberRecipient(privilege *model.Privilege) string {

	// A member who joined as an ActivityPub actor receives ActivityPub deliveries
	if privilege.IdentifierType == model.IdentifierTypeActivityPub {
		return privilege.IdentifierValue
	}

	// RULE: A member who also has an actor receives ActivityPub deliveries, and no email
	identity := model.NewIdentity()

	if err := collections.identityService.LoadByID(collections.session, privilege.IdentityID, &identity); err == nil {
		if identity.HasActivityPubActor() {
			return identity.ActivityPubActor
		}
	}

	if privilege.IdentifierType == model.IdentifierTypeEmail {
		return "mailto:" + privilege.IdentifierValue
	}

	return ""
}
