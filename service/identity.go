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
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"github.com/golang-jwt/jwt/v5"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Identity defines a service that manages all content identitys created and imported by Users.
type Identity struct {
	collection       data.Collection
	activityService  *ActivityStream
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
func (service *Identity) Refresh(collection data.Collection, activityService *ActivityStream, emailService *DomainEmail, jwtService *JWT, privilegeService *Privilege, queue *queue.Queue, host string) {
	service.collection = collection
	service.activityService = activityService
	service.emailService = emailService
	service.jwtService = jwtService
	service.privilegeService = privilegeService
	service.queue = queue
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Identity) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Identity) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Identitys that match the provided criteria
func (service *Identity) Query(criteria exp.Expression, options ...option.Option) ([]model.Identity, error) {
	result := make([]model.Identity, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Identitys that match the provided criteria
func (service *Identity) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Identity records that match the provided criteria
func (service *Identity) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Identity], error) {

	const location = "service.Identity.Range"

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewIdentity), nil
}

// Load retrieves an Identity from the database
func (service *Identity) Load(criteria exp.Expression, identity *model.Identity) error {

	const location = "service.Identity.Load"

	if err := service.collection.Load(notDeleted(criteria), identity); err != nil {
		return derp.Wrap(err, location, "Error loading Identity", criteria)
	}

	return nil
}

// Save adds/updates an Identity in the database
func (service *Identity) Save(identity *model.Identity, note string) error {

	const location = "service.Identity.Save"

	// Fill in missing fields
	if err := service.calcActivityPubActor(identity); err != nil {
		return derp.Wrap(err, location, "Error calculating ActivityPub Actor for Identity")
	}

	// Pick a default name, if necessary
	if err := service.calcName(identity); err != nil {
		return derp.Wrap(err, location, "Error calculating default name for Identity")
	}

	// Validate the value before saving
	if err := service.Schema().Validate(identity); err != nil {
		return derp.Wrap(err, location, "Error validating Identity", identity)
	}

	// Save the identity to the database
	if err := service.collection.Save(identity, note); err != nil {
		return derp.Wrap(err, location, "Error saving Identity", identity, note)
	}

	return nil
}

// Delete removes an Identity from the database (virtual delete)
func (service *Identity) Delete(identity *model.Identity, note string) error {

	// Delete this Identity
	if err := service.collection.Delete(identity, note); err != nil {
		return derp.Wrap(err, "service.Identity.Delete", "Error deleting Identity", identity, note)
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

// New returns a fully initialized model.Identity as a data.Object.
func (service *Identity) ObjectNew() data.Object {
	result := model.NewIdentity()
	return &result
}

func (service *Identity) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Identity); ok {
		return mention.IdentityID
	}

	return primitive.NilObjectID
}

func (service *Identity) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Identity) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewIdentity()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Identity) ObjectSave(object data.Object, comment string) error {
	if identity, ok := object.(*model.Identity); ok {
		return service.Save(identity, comment)
	}
	return derp.InternalError("service.Identity.ObjectSave", "Invalid Object Type", object)
}

func (service *Identity) ObjectDelete(object data.Object, comment string) error {
	if identity, ok := object.(*model.Identity); ok {
		return service.Delete(identity, comment)
	}
	return derp.InternalError("service.Identity.ObjectDelete", "Invalid Object Type", object)
}

func (service *Identity) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Identity.ObjectUserCan", "Not Authorized")
}

func (service *Identity) Schema() schema.Schema {
	return schema.New(model.IdentitySchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID retrieves a single Identity using the provided IdentityID.
func (service *Identity) LoadByID(identityID primitive.ObjectID, identity *model.Identity) error {

	const location = "service.Identity.LoadByID"

	if identityID.IsZero() {
		return derp.BadRequestError(location, "IdentityID cannot be empty", identityID)
	}

	criteria := exp.Equal("_id", identityID)
	return service.Load(criteria, identity)
}

// LoadByToken retrieves a single Identity using the string representation of their IdentityID.
func (service *Identity) LoadByToken(token string, identity *model.Identity) error {

	const location = "service.Identity.LoadByToken"

	identityID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.BadRequestError(location, "Invalid IdentityID", token)
	}

	return service.LoadByID(identityID, identity)
}

// LoadOrCreateByEmail searches for a Guest with the provided emailAddress.
// If a matching record is found, it updates the record with the new values (if necessary).
// If no matching record is found, it creates a new record with the provided values.
func (service *Identity) LoadOrCreate(name string, identifierType string, identifierValue string) (model.Identity, error) {

	const location = "service.Identity.LoadOrCreate"

	// RULE: Identifier Type must be provided
	if identifierType == "" {
		return model.Identity{}, derp.InternalError(location, "Identifier type cannot be empty")
	}

	// RULE: Identifier Value must be provided
	if identifierValue == "" {
		return model.Identity{}, derp.InternalError(location, "Identifier value cannot be empty")
	}

	// Try to load the Identity using the provided identifier
	identity := model.NewIdentity()
	err := service.LoadByIdentifierAndType(identifierType, identifierValue, &identity)

	// If the identity was found, then just return it...
	if err == nil {
		return identity, nil
	}

	// If the error was anything but "not found", then return the error
	if !derp.IsNotFound(err) {
		return model.Identity{}, derp.Wrap(err, location, "Error loading identity", identifierType, identifierValue)
	}

	// Otherwise, populate the identifier into the Identity object
	if ok := identity.SetIdentifier(identifierType, identifierValue); !ok {
		return model.Identity{}, derp.BadRequestError(location, "Invalid Identifier Type", identifierType)
	}

	// Set a default name if the Identity doesn't already have one
	if (identity.Name == "") && (name != "") {
		identity.Name = name
	}

	// Save the Identity to the database
	if err := service.Save(&identity, "Updated"); err != nil {
		return model.Identity{}, derp.Wrap(err, location, "Error saving identity", identity)
	}

	// Done.
	return identity, nil
}

func (service *Identity) LoadByIdentifier(identifierValue string, identity *model.Identity) error {

	const location = "service.identity.LoadByIdentifier"

	// Guess the identifier type
	identifierType := service.GuessIdentifierType(identifierValue)

	if identifierType == "" {
		return derp.BadRequestError(location, "Invalid identifier", identifierValue)
	}

	return service.LoadByIdentifierAndType(identifierType, identifierValue, identity)

}

func (service *Identity) LoadByIdentifierAndType(identifierType string, identifierValue string, identity *model.Identity) error {

	switch identifierType {

	case model.IdentifierTypeEmail:
		return service.LoadByEmailAddress(identifierValue, identity)

	case model.IdentifierTypeActivityPub:
		return service.LoadByActivityPubActor(identifierValue, identity)

	case model.IdentifierTypeWebfinger:
		return service.LoadByWebfingerUsername(identifierValue, identity)
	}

	return derp.InternalError("service.Identity.LoadByAddress", "Invalid Identity Type", identifierType)
}

// LoadByEmail retrieves a single Identity from the database using the provided email address
func (service *Identity) LoadByEmailAddress(emailAddress string, identity *model.Identity) error {
	criteria := exp.Equal("emailAddress", emailAddress)
	return service.Load(criteria, identity)
}

// LoadByActivityPubActor retrieves a single Identity from the database using the provided WebFinger handle
func (service *Identity) LoadByActivityPubActor(actorID string, identity *model.Identity) error {
	criteria := exp.Equal("activityPubActor", actorID)
	return service.Load(criteria, identity)
}

// LoadByWebfingerUsername retrieves a single Identity from the database using the provided WebFinger handle
func (service *Identity) LoadByWebfingerUsername(username string, identity *model.Identity) error {
	criteria := exp.Equal("webfingerUsername", username)
	return service.Load(criteria, identity)
}

// RefreshPrivileges recalculates the privileges for the provided IdentityID by
// loading all privileges and collecting the list of unique CircleIDs and ProductIDs.
func (service *Identity) RefreshPrivileges(identityID primitive.ObjectID) error {

	const location = "service.Identity.RefreshPrivileges"

	// Load the Identity from the database
	identity := model.NewIdentity()
	if err := service.LoadByID(identityID, &identity); err != nil {
		return derp.Wrap(err, location, "Error loading identity", identityID)
	}

	// Get all privileges for this Identity
	privileges, err := service.privilegeService.QueryByIdentity(identityID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading privileges for identity", identityID)
	}

	// Collect the CircleIDs and RemoteProductIDs from each privileges
	privilegeIDs := id.NewSlice()

	for _, privilege := range privileges {

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

	// Save changes
	if err := service.Save(&identity, "Refreshed Privileges"); err != nil {
		return derp.Wrap(err, location, "Error saving identity", identity)
	}

	// Retire in Cabo
	return nil
}

func (service *Identity) SendGuestCode(identity *model.Identity, identifierType string, identifierValue string) error {

	const location = "service.Identity.SendGuestCode"

	// Find the correct sender function based on the identifier type
	var sender func(string, string) error

	switch identifierType {

	case model.IdentifierTypeEmail:
		sender = service.emailService.SendGuestCode

	case model.IdentifierTypeWebfinger, model.IdentifierTypeActivityPub:
		sender = service.sendGuestCode_ActivityPub

	default:
		return derp.BadRequestError(location, "Unrecognized Identifier Type", identifierType)
	}

	// Create a new Guest Code for the identifier :)
	guestCode, err := service.makeGuestCode(nil, identifierType, identifierValue)

	if err != nil {
		return derp.Wrap(err, location, "Error creating Guest Code", identifierValue)
	}

	// Send the Guest Code to the
	if err := sender(identifierValue, guestCode); err != nil {
		return derp.Wrap(err, location, "Error sending Guest Code", identifierValue, guestCode)
	}

	// Looky here. I *am* a Fortunate Son!
	return nil
}

// HasPermissions returns TRUE if the provided identifier has any of the required permissions
func (service *Identity) HasPermissions(identifierType string, identifierValue string, permissions model.Permissions) bool {

	// RULE: If permissions include "anonymous" then anyone can view this item.
	if permissions.IsAnonymous() {
		return true
	}

	// Otherwise, look for the Identity and check it's Privileges
	identity := model.NewIdentity()
	if err := service.LoadByIdentifierAndType(identifierType, identifierValue, &identity); err != nil {
		return false
	}

	// Celebrate good times, come on!
	return identity.PrivilegeIDs.ContainsAny(permissions...)
}

// makeGuestCode creates a new JWT token for the Guest to authenticate
func (service *Identity) makeGuestCode(identity *model.Identity, identifierType string, identifier string) (string, error) {

	// Expires in 1 hour
	expirationDate := time.Now().Add(time.Hour).Unix()

	// Claims for the Identifier, expiring in 1 hour
	claims := jwt.MapClaims{
		"exp": expirationDate, // expiration
		"T":   identifierType, // Identifier Type
		"A":   identifier,     // Identifier (Address)
	}

	// If we have an Identity, then include this in the claims.
	if identity != nil && !identity.IdentityID.IsZero() {
		claims["I"] = identity.IdentityID.Hex() // Identity ID
	}

	// Create and sign the new JWT token
	token, err := service.jwtService.NewToken(claims)

	if err != nil {
		return "", derp.Wrap(err, "service.Identity.makeGuestCode", "Error creating JWT token for Guest Code", identifier)
	}

	// Fantastic.
	return token, nil
}

func (service *Identity) calcActivityPubActor(identity *model.Identity) error {

	const location = "service.Identity.calcActivityPubActor"

	// If we don't have a WebFinger username, then there's nothing to do
	if !identity.HasWebfingerUsername() {
		return nil
	}

	// If we already have an ActivityPub Actor, then we're done
	if identity.HasActivityPubActor() {
		return nil
	}

	// Use Webfinger to look up the ActivityPub Actor.
	record, err := digit.Lookup(identity.WebfingerUsername)

	if err != nil {
		return derp.Wrap(err, location, "Unable to look up WebFinger username", identity.WebfingerUsername)
	}

	// Look for the ActivityPub Actor in the WebFinger record
	for _, link := range record.Links {

		if link.RelationType != digit.RelationTypeSelf {
			continue
		}

		if link.MediaType != vocab.ContentTypeActivityPub {
			continue
		}

		identity.ActivityPubActor = link.Href
		return nil
	}

	// uwuuwuwuuwuwuwuwuwuwuwuwuwuwuwuwuwu
	return derp.BadRequestError(location, "WebFinger record does not include an ActivityPub address", identity.WebfingerUsername)
}

func (service *Identity) calcName(identity *model.Identity) error {

	// If we already have a "Name", then there's nothing else to do
	if identity.Name != "" {
		return nil
	}

	// If we have an ActivityPub Actor, then look up the name from their profile
	if identity.HasActivityPubActor() {

		actor, err := service.activityService.Load(identity.ActivityPubActor, sherlock.AsActor())

		if err != nil {
			return derp.Wrap(err, "service.Identity.calcName", "Error loading ActivityPub Actor", identity.ActivityPubActor)
		}

		identity.Name = actor.Name()
		identity.IconURL = actor.Icon().Href()
		return nil
	}

	// If we have a WebFinger username, then we can use that as the name
	if identity.HasWebfingerUsername() {

		identity.Name = identity.WebfingerUsername
		return nil
	}

	// If we can't look up an ActivityPub actor, then just use the email address as the name
	identity.Name = identity.EmailAddress
	return nil
}

// ParseIdentifier attempts to guess the type of identifier based on its format.
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

func (service *Identity) hostname() string {
	return domain.NameOnly(service.host)
}
