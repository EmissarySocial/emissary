package service

import (
	"iter"
	"net/mail"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"github.com/golang-jwt/jwt/v5"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Identity defines a service that manages all content identitys created and imported by Users.
type Identity struct {
	activityService  actorLoader
	emailService     *DomainEmail
	jwtService       *JWT
	privilegeService *Privilege
	queue            *queue.Queue
	host             string
}

// NewIdentity returns a fully initialized Identity service
func NewIdentity() Identity {
	return Identity{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Identity) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.emailService = factory.Email()
	service.jwtService = factory.JWT()
	service.privilegeService = factory.Privilege()
	service.queue = factory.Queue()
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *Identity) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Identity collection for the provided database session
func (service *Identity) collection(session data.Session) data.Collection {
	return session.Collection("Identity")
}

// Count returns the number of Identity records that match the provided criteria
func (service *Identity) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice of allthe Identitys that match the provided criteria
func (service *Identity) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Identity, error) {
	result := make([]model.Identity, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Identitys that match the provided criteria
func (service *Identity) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Identity records that match the provided criteria
func (service *Identity) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Identity], error) {

	const location = "service.Identity.Range"

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewIdentity), nil
}

// Load retrieves an Identity from the database
func (service *Identity) Load(session data.Session, criteria exp.Expression, identity *model.Identity) error {

	const location = "service.Identity.Load"

	if err := service.collection(session).Load(notDeleted(criteria), identity); err != nil {
		return derp.Wrap(err, location, "Loading Identity", criteria)
	}

	return nil
}

// Save adds/updates an Identity in the database
func (service *Identity) Save(session data.Session, identity *model.Identity, note string) error {

	const location = "service.Identity.Save"

	// Fill in the actor, handle, name, and icon from the ActivityPub profile
	if err := service.calcActorDetails(identity); err != nil {
		return derp.Wrap(err, location, "Calculating actor details for Identity")
	}

	// Pick a default name, if necessary
	calcName(identity)

	// Validate the value before saving
	if _, err := service.Schema().Validate(identity); err != nil {
		return derp.Wrap(err, location, "Validating Identity", identity)
	}

	// Save the identity to the database
	if err := service.collection(session).Save(identity, note); err != nil {
		return derp.Wrap(err, location, "Saving Identity", identity, note)
	}

	// Remove duplicate identifiers from other identities
	if err := service.uniquify(session, identity); err != nil {
		return derp.Wrap(err, location, "Uniquifying Identity", identity)
	}

	// Recalculate the privileges linked to this Identity
	if err := service.privilegeService.refreshIdentity(session, identity); err != nil {
		return derp.Wrap(err, location, "Removing Privileges granted by email", identity)
	}

	// Recalculates privilegeIDs stored in Identity.  This probably duplicates
	// logic from above, but this is a more comprehensive/idempotent calculation.
	if err := service.refreshPrivileges(session, identity); err != nil {
		return derp.Wrap(err, location, "Recalculating privileges for Identity", identity)
	}

	return nil
}

// Delete removes an Identity from the database (virtual delete)
func (service *Identity) Delete(session data.Session, identity *model.Identity, note string) error {

	// Delete this Identity
	if err := service.collection(session).Delete(identity, note); err != nil {
		return derp.Wrap(err, "service.Identity.Delete", "Deleting Identity", identity, note)
	}

	return nil
}

// SaveOrDelete saves the Identity if it is not empty, otherwise deletes it.
// No other business logic is applied to this method.
func (service *Identity) SaveOrDelete(session data.Session, identity *model.Identity, note string) error {

	const location = "service.Identity.SaveOrDelete"

	// If the Identity is empty...
	if identity.IsEmpty() {

		// Delete the identity directly from the datbase (no business logic applied)
		if err := service.collection(session).Delete(identity, note); err != nil {
			return derp.Wrap(err, location, "Deleting empty Identity", identity, note)
		}

		return nil
	}

	// Otherwise, save the Identity directly to the DB (no business logic applied)
	if err := service.collection(session).Save(identity, note); err != nil {
		return derp.Wrap(err, location, "Saving Identity", identity, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Identity) ObjectType() string {
	return "Identity"
}

// ObjectNew returns a fully initialized model.Identity as a data.Object.
func (service *Identity) ObjectNew() data.Object {
	result := model.NewIdentity()
	return &result
}

// ObjectID returns the unique ID of the provided Identity. Implements the ModelService interface.
func (service *Identity) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Identity); ok {
		return mention.IdentityID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Identity that matches the provided criteria. Implements the ModelService interface.
func (service *Identity) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Identity as a data.Object. Implements the ModelService interface.
func (service *Identity) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewIdentity()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Identity in the database. Implements the ModelService interface.
func (service *Identity) ObjectSave(session data.Session, object data.Object, comment string) error {
	if identity, ok := object.(*model.Identity); ok {
		return service.Save(session, identity, comment)
	}
	return derp.Internal("service.Identity.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete marks a Identity as deleted. Implements the ModelService interface.
func (service *Identity) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if identity, ok := object.(*model.Identity); ok {
		return service.Delete(session, identity, comment)
	}
	return derp.Internal("service.Identity.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Identity. Implements the ModelService interface.
func (service *Identity) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Identity.ObjectUserCan", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Identity
func (service *Identity) Schema() schema.Schema {
	return schema.New(model.IdentitySchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID retrieves a single Identity using the provided IdentityID.
func (service *Identity) LoadByID(session data.Session, identityID primitive.ObjectID, identity *model.Identity) error {

	const location = "service.Identity.LoadByID"

	if identityID.IsZero() {
		return derp.BadRequest(location, "IdentityID cannot be empty", identityID)
	}

	criteria := exp.Equal("_id", identityID)
	return service.Load(session, criteria, identity)
}

// LoadByToken retrieves a single Identity using the string representation of their IdentityID.
func (service *Identity) LoadByToken(session data.Session, token string, identity *model.Identity) error {

	const location = "service.Identity.LoadByToken"

	identityID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.BadRequest(location, "Invalid IdentityID", token)
	}

	return service.LoadByID(session, identityID, identity)
}

// LoadOrCreate searches for an Identity with the provided identifier.
// If a matching record is found, it updates the record with the new values (if necessary).
// If no matching record is found, it creates a new record with the provided values.
func (service *Identity) LoadOrCreate(session data.Session, name string, identifierType string, identifierValue string) (model.Identity, error) {

	const location = "service.Identity.LoadOrCreate"

	// RULE: Identifier Type must be provided
	if identifierType == "" {
		return model.Identity{}, derp.Internal(location, "Identifier type cannot be empty")
	}

	// RULE: Identifier Value must be provided
	if identifierValue == "" {
		return model.Identity{}, derp.Internal(location, "Identifier value cannot be empty")
	}

	identity := model.NewIdentity()

	switch identifierType {

	// An email address is its own identifier
	case model.IdentifierTypeEmail:

		found, err := service.locateIdentity(session, exp.Equal("emailAddress", identifierValue), &identity)

		if err != nil {
			return model.Identity{}, derp.Wrap(err, location, "Loading identity by email address", identifierValue)
		}

		if found {
			return identity, nil
		}

		identity.SetIdentifier(model.IdentifierTypeEmail, identifierValue)

	// RULE: A handle or URL is resolved to its actor FIRST, so the lookup is always by actor id
	// and two Identities can never share one actor
	case model.IdentifierTypeWebfinger, model.IdentifierTypeActivityPub:

		actor, err := service.activityService.GetActor(identifierValue)

		if err != nil {
			return model.Identity{}, derp.Wrap(err, location, "Resolving ActivityPub actor", identifierValue)
		}

		found, err := service.locateIdentity(session, exp.Equal("activityPubActor", actor.ID()), &identity)

		if err != nil {
			return model.Identity{}, derp.Wrap(err, location, "Loading identity by actor", actor.ID())
		}

		if found {
			return identity, nil
		}

		applyActor(&identity, actor)

	default:
		return model.Identity{}, derp.BadRequest(location, "Invalid Identifier Type", identifierType)
	}

	// Set a default name if the Identity doesn't already have one
	if (identity.Name == "") && (name != "") {
		identity.Name = name
	}

	// Save the Identity to the database
	if err := service.Save(session, &identity, "Updated"); err != nil {
		return model.Identity{}, derp.Wrap(err, location, "Saving identity", identity)
	}

	// Done.
	return identity, nil
}

// locateIdentity loads the Identity matching the criteria, reporting FALSE (not an error) when none exists
func (service *Identity) locateIdentity(session data.Session, criteria exp.Expression, identity *model.Identity) (bool, error) {

	const location = "service.Identity.locateIdentity"

	if err := service.Load(session, criteria, identity); err != nil {

		// A missing Identity is the "create" half of LoadOrCreate
		if derp.IsNotFound(err) {
			return false, nil
		}

		return false, derp.Wrap(err, location, "Loading Identity", criteria)
	}

	return true, nil
}

// LoadByIdentifier retrieves an Identity using whichever kind of identifier is provided
func (service *Identity) LoadByIdentifier(session data.Session, identifierType string, identifierValue string, identity *model.Identity) error {

	switch identifierType {

	case model.IdentifierTypeEmail:
		return service.LoadByEmailAddress(session, identifierValue, identity)

	case model.IdentifierTypeActivityPub:
		return service.LoadByActivityPubActor(session, identifierValue, identity)

	case model.IdentifierTypeWebfinger:
		return service.LoadByWebfingerUsername(session, identifierValue, identity)
	}

	return derp.Internal("service.Identity.LoadByAddress", "Invalid Identity Type", identifierType)
}

// LoadByEmailAddress retrieves a single Identity from the database using the provided email address
func (service *Identity) LoadByEmailAddress(session data.Session, emailAddress string, identity *model.Identity) error {
	criteria := exp.Equal("emailAddress", emailAddress)
	return service.Load(session, criteria, identity)
}

// LoadByActivityPubActor retrieves a single Identity from the database using the provided WebFinger handle
func (service *Identity) LoadByActivityPubActor(session data.Session, actorID string, identity *model.Identity) error {
	criteria := exp.Equal("activityPubActor", actorID)
	return service.Load(session, criteria, identity)
}

// LoadByWebfingerUsername retrieves a single Identity from the database using the provided WebFinger handle
func (service *Identity) LoadByWebfingerUsername(session data.Session, username string, identity *model.Identity) error {
	criteria := exp.Equal("webfingerUsername", username)
	return service.Load(session, criteria, identity)
}

// RangeByIdentifiers returns an iterator containing all of the Identities that match ANY of the provided identifiers.
func (service *Identity) RangeByIdentifiers(session data.Session, emailAddress string, webfingerUsername string, activityPubActor string) (iter.Seq[model.Identity], error) {

	// RULE: Only identifiers that are present take part. An empty value would match every Identity
	// that lacks that field, which is most of the collection.
	clauses := make([]exp.Expression, 0, 3)

	if emailAddress != "" {
		clauses = append(clauses, exp.Equal("emailAddress", emailAddress))
	}

	if webfingerUsername != "" {
		clauses = append(clauses, exp.Equal("webfingerUsername", webfingerUsername))
	}

	if activityPubActor != "" {
		clauses = append(clauses, exp.Equal("activityPubActor", activityPubActor))
	}

	// Nothing to match means nothing matches
	if len(clauses) == 0 {
		return func(func(model.Identity) bool) { /* nothing to iterate */ }, nil
	}

	// Return query as a RangeFunc
	return service.Range(session, exp.Or(clauses...))
}

// RefreshPrivileges recalculates the privileges for the provided IdentityID by
// loading all privileges and collecting the list of unique CircleIDs and ProductIDs.
func (service *Identity) RefreshPrivileges(session data.Session, identityID primitive.ObjectID) error {

	const location = "service.Identity.RefreshPrivileges"

	// Load the Identity from the database
	identity := model.NewIdentity()
	if err := service.LoadByID(session, identityID, &identity); err != nil {
		return derp.Wrap(err, location, "Loading identity", identityID)
	}

	// Recalculate the privileges for this Identity
	if err := service.refreshPrivileges(session, &identity); err != nil {
		return derp.Wrap(err, location, "Refreshing privileges", identity.IdentityID)
	}

	// Save changes (with no additional business logic)
	if err := service.collection(session).Save(&identity, "Refreshed Privileges"); err != nil {
		return derp.Wrap(err, location, "Saving identity", identity)
	}

	// Retire in Cabo
	return nil
}

// refreshPrivileges recalculates the Circles and Products that this Identity currently has access to
func (service *Identity) refreshPrivileges(session data.Session, identity *model.Identity) error {

	const location = "service.Identity.refreshPrivileges"

	// Get all privileges for this Identity
	privileges, err := service.privilegeService.RangeByIdentity(session, identity.IdentityID)

	if err != nil {
		return derp.Wrap(err, location, "Loading privileges for identity", identity.IdentityID)
	}

	// Collect the CircleIDs and RemoteProductIDs from each privileges
	privilegeIDs := id.NewSlice()

	for privilege := range privileges {

		if !privilege.CircleID.IsZero() {
			privilegeIDs = append(privilegeIDs, privilege.CircleID)
		}

		if !privilege.ProductID.IsZero() {
			privilegeIDs = append(privilegeIDs, privilege.ProductID)
		}
	}

	// Remove Duplicates
	privilegeIDs = slice.Unique(privilegeIDs)

	// Apply to Identity
	identity.PrivilegeIDs = privilegeIDs

	// Success
	return nil
}

// SendGuestCode emails or messages a one-time sign-in code to the provided identifier
func (service *Identity) SendGuestCode(session data.Session, identity *model.Identity, identifierType string, identifierValue string) error {

	const location = "service.Identity.SendGuestCode"

	switch identifierType {

	// Send the Guest Code to an Email Address
	case model.IdentifierTypeEmail:

		guestCode, err := service.makeGuestCode(nil, identifierType, identifierValue)

		if err != nil {
			return derp.Wrap(err, location, "Creating Guest Code", identifierValue)
		}

		if err := service.emailService.SendGuestCode(identifierValue, guestCode); err != nil {
			return derp.Wrap(err, location, "Sending Guest Code", identifierValue, guestCode)
		}

		return nil

	// Resolve the handle or URL to an actor, then send the Guest Code to that actor's inbox
	case model.IdentifierTypeWebfinger, model.IdentifierTypeActivityPub:

		// Load the ActivityPub actor for this identifier
		actor, err := service.activityService.GetActor(identifierValue)

		if err != nil {
			return derp.Wrap(err, location, "Resolving ActivityPub actor", identifierValue)
		}

		// RULE: The code carries the RESOLVED actor id, never the typed handle, so the actor whose
		// inbox receives the code is the actor the Identity is bound to on confirmation.
		guestCode, err := service.makeGuestCode(nil, model.IdentifierTypeActivityPub, actor.ID())

		if err != nil {
			return derp.Wrap(err, location, "Creating Guest Code", actor.ID())
		}

		if err := service.sendGuestCode_ActivityPub(session, identifierValue, actor.ID(), guestCode); err != nil {
			return derp.Wrap(err, location, "Sending Guest Code", identifierValue, guestCode)
		}

		return nil

	}

	// Unrecognized identifier type
	return derp.BadRequest(location, "Unrecognized Identifier Type", identifierType)
}

// HasPermissions returns TRUE if the provided identifier has any of the required permissions
func (service *Identity) HasPermissions(session data.Session, identifierType string, identifierValue string, permissions model.Permissions) bool {

	// RULE: If permissions include "anonymous" then anyone can view this item.
	if permissions.IsAnonymous() {
		return true
	}

	// Otherwise, look for the Identity and check it's Privileges
	identity := model.NewIdentity()
	if err := service.LoadByIdentifier(session, identifierType, identifierValue, &identity); err != nil {
		return false
	}

	// Celebrate good times, come on!
	return identity.PrivilegeIDs.ContainsAny(permissions...)
}

// GuessIdentifierType attempts to guess the type of identifier based on its format.
func (service *Identity) GuessIdentifierType(identifier string) string {

	// WebFinger begins with "@" and needs to be translated into an ActivityPub Actor
	if strings.HasPrefix(identifier, "@") {

		identifier = strings.TrimPrefix(identifier, "@")
		if _, err := mail.ParseAddress(identifier); err == nil {
			return model.IdentifierTypeWebfinger
		}

		// Otherwise, failure.
		return ""
	}

	// ActivityPub Actor URLs begin with "https://" or "http://"
	if strings.HasPrefix(identifier, "https://") || strings.HasPrefix(identifier, "http://") {
		return model.IdentifierTypeActivityPub
	}

	// Assume Email Address
	if _, err := mail.ParseAddress(identifier); err == nil {
		return model.IdentifierTypeEmail
	}

	// Unknown identifier type
	return ""
}

// makeGuestCode creates a new JWT token for the Guest to authenticate
func (service *Identity) makeGuestCode(identity *model.Identity, identifierType string, identifier string) (string, error) {

	const location = "service.Identity.makeGuestCode"

	// Claims for the Identifier, expiring in 1 hour
	claims := guestCodeClaims(identity, identifierType, identifier, time.Now().Add(time.Hour).Unix())

	// Create and sign the new JWT token
	token, err := service.jwtService.NewToken(claims)

	if err != nil {
		return "", derp.Wrap(err, location, "Creating JWT token for Guest Code", identifier)
	}

	// Fantastic.
	return token, nil
}

// guestCodeClaims builds the JWT claims that a guest sign-in code carries
func guestCodeClaims(identity *model.Identity, identifierType string, identifier string, expires int64) jwt.MapClaims {

	claims := jwt.MapClaims{
		"exp": expires,        // expiration
		"T":   identifierType, // Identifier Type
		"A":   identifier,     // Identifier (Address)
	}

	// An Identity that already exists rides along so the code re-attaches to it
	if identity == nil {
		return claims
	}

	if identity.IdentityID.IsZero() {
		return claims
	}

	claims["I"] = identity.IdentityID.Hex() // Identity ID
	return claims
}

// calcActorDetails fills this Identity's actor, handle, name, and icon from its ActivityPub profile
func (service *Identity) calcActorDetails(identity *model.Identity) error {

	const location = "service.Identity.calcActorDetails"

	// A handle that is not yet bound to an actor is resolved here (legacy rows, WEBFINGER identifiers)
	if identity.NotHasActivityPubActor() {

		if identity.NotHasWebfingerUsername() {
			return nil
		}

		actor, err := service.activityService.GetActor(identity.WebfingerUsername)

		if err != nil {
			return derp.Wrap(err, location, "Resolving WebFinger username", identity.WebfingerUsername)
		}

		applyActor(identity, actor)
		return nil
	}

	// RULE: A routine save stays off the network. The actor is loaded only while a detail is missing.
	if identity.HasWebfingerUsername() && (identity.Name != "") {
		return nil
	}

	// Otherwise, load the actor from the network to confirm the Webfinger results are not malicious
	actor, err := service.activityService.GetActor(identity.ActivityPubActor)

	if err != nil {
		return derp.Wrap(err, location, "Loading ActivityPub Actor", identity.ActivityPubActor)
	}

	applyActor(identity, actor)
	return nil
}

// applyActor copies an actor's id, handle, name, and icon onto the Identity, keeping a name and icon already set.
// This prevents a malicious WebFinger result from fraudulently claiming ownership of an account.
func applyActor(identity *model.Identity, actor streams.Document) {

	identity.ActivityPubActor = actor.ID()
	identity.WebfingerUsername = actorHandle(actor)

	if identity.Name == "" {
		identity.Name = actor.Name()
	}

	if identity.IconURL == "" {
		identity.IconURL = actor.Icon().Href()
	}
}

// actorHandle returns the actor's own @username@host handle, or empty if the actor has no preferredUsername
func actorHandle(actor streams.Document) string {

	// UsernameOrID falls back to the bare id, which is a URL, not a handle
	if handle := actor.UsernameOrID(); strings.HasPrefix(handle, "@") {
		return handle
	}

	return ""
}

// calcName picks a default display name for an Identity that has none
func calcName(identity *model.Identity) {

	if identity.Name != "" {
		return
	}

	if identity.HasWebfingerUsername() {
		identity.Name = identity.WebfingerUsername
		return
	}

	identity.Name = identity.EmailAddress
}

// uniquify takes duplicate Identitifiers from any other Identities
// and merges their privileges with the provided Identity.
func (service *Identity) uniquify(session data.Session, identity *model.Identity) error {

	const location = "service.Identity.uniquify"

	// Take Privileges that match my Identifiers from other
	privileges, err := service.privilegeService.RangeByIdentifiers(session, identity.EmailAddress, identity.WebfingerUsername, identity.ActivityPubActor)

	if err != nil {
		return derp.Wrap(err, location, "Loading Privileges", identity)
	}

	// Link the current identity to each Privilege listed
	for privilege := range privileges {

		if err := service.privilegeService.maybeSetIdentity(session, &privilege, identity); err != nil {
			return derp.Wrap(err, location, "Linking Identity to this Privilege", privilege)
		}
	}

	// Remove my Identifiers from other Identities
	identities, err := service.RangeByIdentifiers(session, identity.EmailAddress, identity.WebfingerUsername, identity.ActivityPubActor)

	if err != nil {
		return derp.Wrap(err, location, "Loading Identities", identity)
	}

	for other := range identities {

		// RULE: No need to change myself
		if other.IdentityID == identity.IdentityID {
			continue
		}

		other.RemoveIdentifier(model.IdentifierTypeEmail, identity.EmailAddress)
		other.RemoveIdentifier(model.IdentifierTypeWebfinger, identity.WebfingerUsername)
		other.RemoveIdentifier(model.IdentifierTypeActivityPub, identity.ActivityPubActor)

		if err := service.SaveOrDelete(session, &other, "Removed identity"); err != nil {
			derp.Report(derp.Wrap(err, location, "Uniquifying Identity", other))
		}
	}

	return nil
}

// hostname returns the bare hostname of the Domain that owns this Identity
func (service *Identity) hostname() string {
	return uri.Hostname(service.host)
}
