package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests use hand-built data.Collection fakes for the same reason follower_test.go does:
// data-mock matches on the raw bson tag string and cannot see `deleteDate` inside the inlined
// journal.Journal that every notDeleted() criteria queries.

/******************************************
 * In-Memory Fakes
 ******************************************/

// stubActorLoader is an actorLoader that answers from a fixed map of address to actor document,
// counting every call. An address it does not know is "not found".
type stubActorLoader struct {
	actors map[string]streams.Document
	calls  int
}

// GetActor implements actorLoader, returning the canned document for the address
func (loader *stubActorLoader) GetActor(address string) (streams.Document, error) {

	loader.calls++

	if actor, ok := loader.actors[address]; ok {
		return actor, nil
	}

	return streams.NilDocument(), derp.NotFound("test", "unknown actor", address)
}

// newActor builds a minimal Person document. An empty username yields an actor with no handle.
func newActor(id string, username string, name string) streams.Document {

	return streams.NewDocument(mapof.Any{
		"id":                id,
		"type":              "Person",
		"preferredUsername": username,
		"name":              name,
		"icon": mapof.Any{
			"type": "Image",
			"href": id + "/icon.png",
		},
	})
}

// identityFakeCollection is an in-memory data.Collection that holds model.Identity records
type identityFakeCollection struct {
	records []model.Identity
}

// Context implements the data.Collection interface, returning a background context
func (c *identityFakeCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *identityFakeCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *identityFakeCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator returns every matching Identity, in insertion order
func (c *identityFakeCollection) Iterator(criteria exp.Expression, _ ...option.Option) (data.Iterator, error) {

	result := make([]model.Identity, 0)

	for _, record := range c.records {
		if matchesIdentity(criteria, record) {
			result = append(result, record)
		}
	}

	return &identityFakeIterator[model.Identity]{records: result}, nil
}

// Load copies the first matching Identity into the target
func (c *identityFakeCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	for _, record := range c.records {

		if !matchesIdentity(criteria, record) {
			continue
		}

		identity, ok := target.(*model.Identity)

		if !ok {
			return derp.Internal("test", "unexpected target type")
		}

		*identity = record
		return nil
	}

	return derp.NotFound("test", "not found")
}

// Save upserts an Identity by its IdentityID
func (c *identityFakeCollection) Save(object data.Object, _ string) error {

	identity, ok := object.(*model.Identity)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	for index, record := range c.records {
		if record.IdentityID == identity.IdentityID {
			c.records[index] = *identity
			return nil
		}
	}

	c.records = append(c.records, *identity)
	return nil
}

// Delete writes the Identity back with its delete date stamped, the way the Mongo driver's soft delete does
func (c *identityFakeCollection) Delete(object data.Object, _ string) error {

	identity, ok := object.(*model.Identity)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	for index, record := range c.records {
		if record.IdentityID == identity.IdentityID {
			c.records[index] = *identity
			c.records[index].DeleteDate = 1
			return nil
		}
	}

	return nil
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *identityFakeCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

// find returns the stored copy of an Identity by its ID
func (c *identityFakeCollection) find(t *testing.T, identityID primitive.ObjectID) model.Identity {

	t.Helper()

	for _, record := range c.records {
		if record.IdentityID == identityID {
			return record
		}
	}

	require.FailNow(t, "identity not found", identityID.Hex())
	return model.Identity{}
}

// matchesIdentity reports whether an Identity satisfies a criteria on _id, emailAddress,
// webfingerUsername, activityPubActor, or deleteDate. Anything else conservatively does not match.
func matchesIdentity(criteria exp.Expression, record model.Identity) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.IdentityID == value

		case "emailAddress":
			value, ok := predicate.Value.(string)
			return ok && record.EmailAddress == value

		case "webfingerUsername":
			value, ok := predicate.Value.(string)
			return ok && record.WebfingerUsername == value

		case "activityPubActor":
			value, ok := predicate.Value.(string)
			return ok && record.ActivityPubActor == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// privilegeFakeCollection is an in-memory data.Collection that holds model.Privilege records
type privilegeFakeCollection struct {
	records []model.Privilege
}

// Context implements the data.Collection interface, returning a background context
func (c *privilegeFakeCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *privilegeFakeCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *privilegeFakeCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator returns every matching Privilege, in insertion order
func (c *privilegeFakeCollection) Iterator(criteria exp.Expression, _ ...option.Option) (data.Iterator, error) {

	result := make([]model.Privilege, 0)

	for _, record := range c.records {
		if matchesPrivilege(criteria, record) {
			result = append(result, record)
		}
	}

	return &identityFakeIterator[model.Privilege]{records: result}, nil
}

// Load copies the first matching Privilege into the target
func (c *privilegeFakeCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	for _, record := range c.records {

		if !matchesPrivilege(criteria, record) {
			continue
		}

		privilege, ok := target.(*model.Privilege)

		if !ok {
			return derp.Internal("test", "unexpected target type")
		}

		*privilege = record
		return nil
	}

	return derp.NotFound("test", "not found")
}

// Save upserts a Privilege by its PrivilegeID
func (c *privilegeFakeCollection) Save(object data.Object, _ string) error {

	privilege, ok := object.(*model.Privilege)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	for index, record := range c.records {
		if record.PrivilegeID == privilege.PrivilegeID {
			c.records[index] = *privilege
			return nil
		}
	}

	c.records = append(c.records, *privilege)
	return nil
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *privilegeFakeCollection) Delete(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *privilegeFakeCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

// matchesPrivilege reports whether a Privilege satisfies a criteria on _id, identityId,
// identifierType, identifierValue, or deleteDate. Anything else conservatively does not match.
func matchesPrivilege(criteria exp.Expression, record model.Privilege) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.PrivilegeID == value

		case "identityId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.IdentityID == value

		case "identifierType":
			value, ok := predicate.Value.(string)
			return ok && record.IdentifierType == value

		case "identifierValue":
			value, ok := predicate.Value.(string)
			return ok && record.IdentifierValue == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// identityFakeIterator walks a fixed slice of records. Implements data.Iterator.
type identityFakeIterator[T any] struct {
	records []T
	index   int
}

// Next copies the next record into the target, returning FALSE when the list is exhausted
func (it *identityFakeIterator[T]) Next(target any) bool {

	if it.index >= len(it.records) {
		return false
	}

	typed, ok := target.(*T)

	if !ok {
		return false
	}

	*typed = it.records[it.index]
	it.index++

	return true
}

// Count returns the number of records this iterator walks
func (it *identityFakeIterator[T]) Count() int { return len(it.records) }

// Close releases this iterator. It holds nothing.
func (it *identityFakeIterator[T]) Close() error { return nil }

// Error returns the error encountered while iterating, of which there are none
func (it *identityFakeIterator[T]) Error() error { return nil }

// identityFakeSession hands out the Identity and Privilege fakes by collection name, and carries
// a post-commit spool so guest-code sends can be inspected instead of queued
type identityFakeSession struct {
	identities *identityFakeCollection
	privileges *privilegeFakeCollection
	ctx        context.Context
}

// Collection implements the data.Session interface
func (s identityFakeSession) Collection(name string) data.Collection {

	if name == "Privilege" {
		return s.privileges
	}

	return s.identities
}

// Context implements the data.Session interface, carrying the post-commit spool
func (s identityFakeSession) Context() context.Context { return s.ctx }

// Close implements the data.Session interface. The stub holds no resources to release.
func (s identityFakeSession) Close() {}

// newIdentityTestService returns an Identity service backed by in-memory fakes and the given actors
func newIdentityTestService(loader *stubActorLoader) (*Identity, identityFakeSession) {

	privilegeService := NewPrivilege()

	jwtService := NewJWT()
	jwtService.Refresh(mockdb.New())

	service := &Identity{
		activityService:  loader,
		jwtService:       &jwtService,
		privilegeService: &privilegeService,
		host:             "https://a.example",
	}

	session := identityFakeSession{
		identities: &identityFakeCollection{},
		privileges: &privilegeFakeCollection{},
		ctx:        postcommit.WithContext(context.Background(), postcommit.NewTasks()),
	}

	return service, session
}

// bobLoader returns a loader that knows Bob under his handle and his actor id
func bobLoader() *stubActorLoader {

	bob := newActor("https://good.example/@bob", "bob", "Bob")

	return &stubActorLoader{
		actors: map[string]streams.Document{
			"@bob@good.example":         bob,
			"https://good.example/@bob": bob,
		},
	}
}

/******************************************
 * Guest Codes (Task 1)
 ******************************************/

// TestGuestCodeClaims_ActivityPubCarriesActorID pins the shape of the claims a guest code carries
func TestGuestCodeClaims_ActivityPubCarriesActorID(t *testing.T) {

	claims := guestCodeClaims(nil, model.IdentifierTypeActivityPub, "https://good.example/@bob", 12345)

	require.Equal(t, model.IdentifierTypeActivityPub, claims["T"])
	require.Equal(t, "https://good.example/@bob", claims["A"])
	require.Equal(t, int64(12345), claims["exp"])
	require.NotContains(t, claims, "I")

	// An existing Identity rides along by ID
	identity := model.NewIdentity()
	claims = guestCodeClaims(&identity, model.IdentifierTypeEmail, "sarah@example.com", 1)
	require.Equal(t, identity.IdentityID.Hex(), claims["I"])

	// A zero-valued Identity does not
	zero := model.Identity{}
	claims = guestCodeClaims(&zero, model.IdentifierTypeEmail, "sarah@example.com", 1)
	require.NotContains(t, claims, "I")
}

// TestSendGuestCode_ResolvesOnce is BUG-01's invariant I1 stated directly: the handle is resolved
// exactly once, the code goes to that actor, and the code names the actor id, never the handle.
func TestSendGuestCode_ResolvesOnce(t *testing.T) {

	const alice = "https://evil.example/@alice"

	loader := &stubActorLoader{
		actors: map[string]streams.Document{
			"@alice@evil.example": newActor(alice, "alice", "Alice"),
		},
	}

	service, session := newIdentityTestService(loader)

	require.NoError(t, service.SendGuestCode(session, nil, model.IdentifierTypeWebfinger, "@alice@evil.example"))
	require.Equal(t, 1, loader.calls, "the handle is resolved exactly once")

	// The message was spooled, addressed to the resolved actor
	tasks := postcommit.From(session.Context()).Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, []string{alice}, tasks[0].Arguments["to"])

	object, ok := tasks[0].Arguments["object"].(mapof.Any)
	require.True(t, ok, "the activity carries its Note as a map")

	content, ok := object["content"].(string)
	require.True(t, ok)

	// The code inside the message carries the RESOLVED actor id, not the typed handle
	match := regexp.MustCompile(`/@guest/signin/([A-Za-z0-9._-]+)`).FindStringSubmatch(content)
	require.Len(t, match, 2, "the message links to a guest code")

	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(match[1], claims)
	require.NoError(t, err)
	require.Equal(t, model.IdentifierTypeActivityPub, claims["T"])
	require.Equal(t, alice, claims["A"])
}

// TestSendGuestCode_UnknownActorFails pins that an unresolvable handle is reported, not sent
func TestSendGuestCode_UnknownActorFails(t *testing.T) {

	loader := &stubActorLoader{actors: map[string]streams.Document{}}
	service, session := newIdentityTestService(loader)

	err := service.SendGuestCode(session, nil, model.IdentifierTypeWebfinger, "@nobody@nowhere.example")
	require.Error(t, err)
	require.Empty(t, postcommit.From(session.Context()).Drain())

	// An unrecognized identifier type never reaches the network
	require.Error(t, service.SendGuestCode(session, nil, "BOGUS", "whatever"))
	require.Equal(t, 1, loader.calls)
}

/******************************************
 * LoadOrCreate (Task 2)
 ******************************************/

// TestLoadOrCreate_HandleFindsExistingActorIdentity pins that a handle lookup lands on the Identity
// already bound to that actor, instead of creating a second one keyed on the handle
func TestLoadOrCreate_HandleFindsExistingActorIdentity(t *testing.T) {

	service, session := newIdentityTestService(bobLoader())

	existing := model.NewIdentity()
	existing.ActivityPubActor = "https://good.example/@bob"
	existing.WebfingerUsername = "@bob@good.example"
	existing.Name = "Bob"
	session.identities.records = append(session.identities.records, existing)

	identity, err := service.LoadOrCreate(session, "", model.IdentifierTypeWebfinger, "@bob@good.example")
	require.NoError(t, err)
	require.Equal(t, existing.IdentityID, identity.IdentityID)
	require.Len(t, session.identities.records, 1)

	// The same by actor URL
	identity, err = service.LoadOrCreate(session, "", model.IdentifierTypeActivityPub, "https://good.example/@bob")
	require.NoError(t, err)
	require.Equal(t, existing.IdentityID, identity.IdentityID)
	require.Len(t, session.identities.records, 1)
}

// TestLoadOrCreate_TwoHandlesOneActorOneIdentity is the vanity-domain case: two handles that resolve
// to one actor share one Identity, and the stored handle is the actor's own (decision D3)
func TestLoadOrCreate_TwoHandlesOneActorOneIdentity(t *testing.T) {

	ben := newActor("https://emissary.example/@ben", "ben", "Ben")

	loader := &stubActorLoader{
		actors: map[string]streams.Document{
			"@ben@pate.example":             ben,
			"@ben@emissary.example":         ben,
			"https://emissary.example/@ben": ben,
		},
	}

	service, session := newIdentityTestService(loader)

	first, err := service.LoadOrCreate(session, "", model.IdentifierTypeWebfinger, "@ben@pate.example")
	require.NoError(t, err)

	second, err := service.LoadOrCreate(session, "", model.IdentifierTypeWebfinger, "@ben@emissary.example")
	require.NoError(t, err)

	require.Equal(t, first.IdentityID, second.IdentityID)
	require.Len(t, session.identities.records, 1)
	require.Equal(t, "https://emissary.example/@ben", first.ActivityPubActor)
	require.Equal(t, "@ben@emissary.example", first.WebfingerUsername, "the stored handle is the actor's own, not the vanity one")
	require.Equal(t, "Ben", first.Name)
	require.Equal(t, "https://emissary.example/@ben/icon.png", first.IconURL)
}

// TestLoadOrCreate_EmailAndErrors pins the unchanged email path and the guards
func TestLoadOrCreate_EmailAndErrors(t *testing.T) {

	service, session := newIdentityTestService(bobLoader())

	identity, err := service.LoadOrCreate(session, "Sarah", model.IdentifierTypeEmail, "sarah@example.com")
	require.NoError(t, err)
	require.Equal(t, "sarah@example.com", identity.EmailAddress)
	require.Equal(t, "Sarah", identity.Name)

	again, err := service.LoadOrCreate(session, "Other", model.IdentifierTypeEmail, "sarah@example.com")
	require.NoError(t, err)
	require.Equal(t, identity.IdentityID, again.IdentityID)
	require.Len(t, session.identities.records, 1)

	_, err = service.LoadOrCreate(session, "", "", "sarah@example.com")
	require.Error(t, err)

	_, err = service.LoadOrCreate(session, "", model.IdentifierTypeEmail, "")
	require.Error(t, err)

	_, err = service.LoadOrCreate(session, "", "BOGUS", "value")
	require.Error(t, err)

	_, err = service.LoadOrCreate(session, "", model.IdentifierTypeWebfinger, "@nobody@nowhere.example")
	require.Error(t, err)
	require.Len(t, session.identities.records, 1, "an unresolvable handle creates nothing")
}

/******************************************
 * Save derives from the actor (Task 3)
 ******************************************/

// TestSave_DerivesHandleFromActor pins that binding an actor fills the handle, name, and icon from
// the actor document, and that a routine save afterwards stays off the network
func TestSave_DerivesHandleFromActor(t *testing.T) {

	loader := bobLoader()
	service, session := newIdentityTestService(loader)

	identity := model.NewIdentity()
	identity.SetIdentifier(model.IdentifierTypeActivityPub, "https://good.example/@bob")

	require.NoError(t, service.Save(session, &identity, "test"))
	require.Equal(t, "@bob@good.example", identity.WebfingerUsername)
	require.Equal(t, "Bob", identity.Name)
	require.Equal(t, "https://good.example/@bob/icon.png", identity.IconURL)
	require.Equal(t, 1, loader.calls)

	// Nothing is missing now, so a second save does not load the actor again
	require.NoError(t, service.Save(session, &identity, "again"))
	require.Equal(t, 1, loader.calls)

	// A name the guest chose is kept
	identity.Name = "Robert"
	identity.SetIdentifier(model.IdentifierTypeActivityPub, "https://good.example/@bob")
	require.NoError(t, service.Save(session, &identity, "renamed"))
	require.Equal(t, "Robert", identity.Name)
	require.Equal(t, "@bob@good.example", identity.WebfingerUsername)
}

// TestSave_ActorWithoutUsernameLeavesHandleEmpty pins the degradation for actors that carry no
// preferredUsername: no handle, but still a real Identity
func TestSave_ActorWithoutUsernameLeavesHandleEmpty(t *testing.T) {

	const blog = "https://blog.example/actor"

	loader := &stubActorLoader{
		actors: map[string]streams.Document{
			blog: newActor(blog, "", "The Blog"),
		},
	}

	service, session := newIdentityTestService(loader)

	identity := model.NewIdentity()
	identity.SetIdentifier(model.IdentifierTypeActivityPub, blog)

	require.NoError(t, service.Save(session, &identity, "test"))
	require.Equal(t, "", identity.WebfingerUsername)
	require.Equal(t, "The Blog", identity.Name)
	require.False(t, identity.IsEmpty())
	require.Zero(t, session.identities.find(t, identity.IdentityID).DeleteDate)
}

// TestSave_LegacyHandleResolvesThroughActorLoader pins that a row holding only a handle is bound
// to its actor through the shared actor loader, and re-labelled with the actor's own handle
func TestSave_LegacyHandleResolvesThroughActorLoader(t *testing.T) {

	ben := newActor("https://emissary.example/@ben", "ben", "Ben")

	loader := &stubActorLoader{
		actors: map[string]streams.Document{
			"@ben@pate.example":             ben,
			"https://emissary.example/@ben": ben,
		},
	}

	service, session := newIdentityTestService(loader)

	identity := model.NewIdentity()
	identity.WebfingerUsername = "@ben@pate.example"

	require.NoError(t, service.Save(session, &identity, "test"))
	require.Equal(t, "https://emissary.example/@ben", identity.ActivityPubActor)
	require.Equal(t, "@ben@emissary.example", identity.WebfingerUsername)
	require.Equal(t, "Ben", identity.Name)

	// An unresolvable handle fails the save rather than storing a half-bound Identity
	broken := model.NewIdentity()
	broken.WebfingerUsername = "@nobody@nowhere.example"
	require.Error(t, service.Save(session, &broken, "test"))
	require.Len(t, session.identities.records, 1)
}

// TestSave_EmailOnlyStaysOffTheNetwork pins that an email Identity never touches the actor loader
func TestSave_EmailOnlyStaysOffTheNetwork(t *testing.T) {

	loader := bobLoader()
	service, session := newIdentityTestService(loader)

	identity := model.NewIdentity()
	identity.EmailAddress = "sarah@example.com"

	require.NoError(t, service.Save(session, &identity, "test"))
	require.Equal(t, "sarah@example.com", identity.Name)
	require.Equal(t, 0, loader.calls)
}

/******************************************
 * One actor, one Identity (Task 4)
 ******************************************/

// TestUniquify_MovesActorFromOtherIdentity pins that proving control of an actor takes that actor,
// and every Privilege keyed on it, away from any other Identity that held it
func TestUniquify_MovesActorFromOtherIdentity(t *testing.T) {

	const bob = "https://good.example/@bob"

	service, session := newIdentityTestService(bobLoader())

	// A legacy row holds Bob's actor with no handle, plus a Privilege keyed on the actor
	circleID := primitive.NewObjectID()

	legacy := model.NewIdentity()
	legacy.ActivityPubActor = bob
	legacy.Name = "Bob"
	legacy.PrivilegeIDs = append(legacy.PrivilegeIDs, circleID)
	session.identities.records = append(session.identities.records, legacy)

	privilege := model.NewPrivilege()
	privilege.IdentityID = legacy.IdentityID
	privilege.IdentifierType = model.IdentifierTypeActivityPub
	privilege.IdentifierValue = bob
	privilege.CircleID = circleID
	session.privileges.records = append(session.privileges.records, privilege)

	// An email Identity proves control of Bob's inbox and binds the actor
	claimant := model.NewIdentity()
	claimant.EmailAddress = "bob@mail.example"
	claimant.Name = "Bob by email"
	claimant.SetIdentifier(model.IdentifierTypeActivityPub, bob)

	require.NoError(t, service.Save(session, &claimant, "test"))

	// The Privilege now belongs to the claimant
	require.Equal(t, claimant.IdentityID, session.privileges.records[0].IdentityID)
	require.Contains(t, claimant.PrivilegeIDs, circleID)

	// The legacy row lost the actor, and having nothing else, was deleted
	after := session.identities.find(t, legacy.IdentityID)
	require.Equal(t, "", after.ActivityPubActor)
	require.NotZero(t, after.DeleteDate)

	// Exactly one live Identity carries the actor
	live := 0

	for _, record := range session.identities.records {

		if record.DeleteDate != 0 {
			continue
		}

		if record.ActivityPubActor == bob {
			live++
		}
	}

	require.Equal(t, 1, live)
}

// TestRangeByIdentifiers_SkipsEmptyValues pins decision D5: a blank identifier matches nothing,
// rather than every Identity that lacks that field
func TestRangeByIdentifiers_SkipsEmptyValues(t *testing.T) {

	service, session := newIdentityTestService(bobLoader())

	byEmail := model.NewIdentity()
	byEmail.EmailAddress = "a@example.com"

	byActor := model.NewIdentity()
	byActor.ActivityPubActor = "https://good.example/@bob"

	session.identities.records = append(session.identities.records, byEmail, byActor)

	// An actor nobody has, with blank email and handle, matches nothing
	found := collectIdentities(t, service, session, "", "", "https://nobody.example/@x")
	require.Empty(t, found)

	// All blank matches nothing, without error
	found = collectIdentities(t, service, session, "", "", "")
	require.Empty(t, found)

	// A present identifier still matches its own record only
	found = collectIdentities(t, service, session, "a@example.com", "", "")
	require.Len(t, found, 1)
	require.Equal(t, byEmail.IdentityID, found[0].IdentityID)

	found = collectIdentities(t, service, session, "", "", "https://good.example/@bob")
	require.Len(t, found, 1)
	require.Equal(t, byActor.IdentityID, found[0].IdentityID)
}

// collectIdentities drains RangeByIdentifiers into a slice
func collectIdentities(t *testing.T, service *Identity, session data.Session, email string, handle string, actor string) []model.Identity {

	t.Helper()

	seq, err := service.RangeByIdentifiers(session, email, handle, actor)
	require.NoError(t, err)

	result := make([]model.Identity, 0)

	for identity := range seq {
		result = append(result, identity)
	}

	return result
}
