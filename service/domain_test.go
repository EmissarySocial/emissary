package service

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/******************************************
 * Untested Paths
 *
 * createOwner and inviteOwner inside bootstrap need a fully wired
 * User service and Steranko, so no test here sets CreateOwner.
 * persist's second Validate cannot fail alone, because Schema()
 * returns the same schema as the first.  Start's background
 * upgrade needs a live database; queries tests UpgradeMongoDB.
 ******************************************/

// domainTestConnection is the local MongoDB replica set that the integration tests require
const domainTestConnection = "mongodb://localhost:27017/?directConnection=true"

// newTestDomainService returns a Domain service for hostname whose record lives in an in-memory
// database.  Its background upgrade stops at once, because there is no real database to upgrade.
func newTestDomainService(t *testing.T, hostname string) (*Domain, data.Session) {

	t.Helper()

	server := mockdb.New()
	session, err := server.Session(context.Background())
	require.NoError(t, err)

	service := NewDomain()
	service.hostname = hostname
	service.configuration = config.Domain{Hostname: hostname, Label: "Test Domain"}
	service.database = func() *mongo.Database { return nil }
	service.withTransaction = server.WithTransaction
	service.newSession = func(timeout time.Duration) (data.Session, context.CancelFunc, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		session, err := server.Session(ctx)
		return session, cancel, err
	}

	return &service, session
}

// storeTestDomain writes a Domain record for hostname straight to the database
func storeTestDomain(t *testing.T, session data.Session, hostname string) model.WritableDomain {

	t.Helper()

	domain := model.NewWritableDomain()
	domain.Hostname = hostname
	domain.Label = "Stored Label"
	require.NoError(t, session.Collection("Domain").Save(&domain, "Stored"))

	return domain
}

// loadStoredDomain returns the Domain record stored in the database that session reaches
func loadStoredDomain(t *testing.T, session data.Session) model.WritableDomain {

	t.Helper()

	result := model.NewWritableDomain()
	require.NoError(t, session.Collection("Domain").Load(exp.All(), &result))

	return result
}

// requireNothingStored fails unless the database that session reaches holds no Domain record
func requireNothingStored(t *testing.T, session data.Session) {

	t.Helper()

	result := model.NewWritableDomain()
	err := session.Collection("Domain").Load(exp.All(), &result)
	require.True(t, derp.IsNotFound(err), "expected no Domain record, got %v", err)
}

// newDomainTestDatabase returns a throwaway database on the local MongoDB replica set, dropped when
// the test ends.  The test is skipped in -short mode, or when no replica set is reachable.
func newDomainTestDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(domainTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + domainTestConnection)
	}

	// RULE: Change streams need a replica set, and a standalone server names none
	hello := bson.M{}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil || hello["setName"] == nil {
		_ = client.Disconnect(ctx) // Nothing to report: the test is skipped either way
		t.Skip("Skipping: no MongoDB replica set at " + domainTestConnection)
	}

	database := client.Database("emissary_domaintest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)     // A leftover test database is harmless
		_ = client.Disconnect(ctx) // ...and so is a connection closed by process exit
	})

	return database
}

// newReplicaSetSession returns a session on a throwaway database in the local replica set, and the
// server it came from, for tests that need a real BSON round trip (see AGENTS.md).
func newReplicaSetSession(t *testing.T) (data.Server, data.Session) {

	t.Helper()

	server := mongodb.NewServer(newDomainTestDatabase(t))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	session, err := server.Session(ctx)
	require.NoError(t, err)

	return server, session
}

// requireSharesNoMapOrSlice fails if any map or slice field of a and b points at the same memory.
// A nil field proves nothing, so the fixtures used with it must populate every field.
func requireSharesNoMapOrSlice(t *testing.T, a model.Domain, b model.Domain) {

	t.Helper()

	aValue := reflect.ValueOf(a)
	bValue := reflect.ValueOf(b)

	for index := range aValue.NumField() {

		field := aValue.Type().Field(index)

		if kind := field.Type.Kind(); kind != reflect.Map && kind != reflect.Slice {
			continue
		}

		require.False(t, aValue.Field(index).IsNil(), "the fixture must populate %s", field.Name)
		require.False(t, bValue.Field(index).IsNil(), "the fixture must populate %s", field.Name)
		require.NotEqual(t, aValue.Field(index).Pointer(), bValue.Field(index).Pointer(), "%s must not be shared", field.Name)
	}
}

// newPopulatedDomain returns a Domain with a value in every map and slice, so that a test which
// compares map and slice pointers has something to compare.
func newPopulatedDomain(hostname string) model.WritableDomain {

	domain := model.NewWritableDomain()
	domain.Hostname = hostname
	domain.Label = "Populated"
	domain.Data["sso_secret"] = "secret"
	domain.ThemeData["stylesheet"] = "body {}"
	domain.RegistrationData["field"] = "value"
	domain.Syndication = sliceof.Object[form.LookupCode]{{Value: "bluesky", Label: "Bluesky"}}
	domain.StartupTasks = sliceof.String{"/startup/content"}
	domain.MLSMode = model.DomainMLSModeGroups
	domain.MLSGroupIDs = sliceof.String{"group"}
	domain.Connections["stripe"] = model.Connection{ProviderID: "stripe", Type: "payment", Data: mapof.NewAny()}

	return domain
}

// failingSession is a data.Session whose every collection fails to load
type failingSession struct {
	data.Session
}

// Collection returns a collection whose Load always fails
func (session failingSession) Collection(_ string) data.Collection {
	return failingCollection{}
}

// failingCollection is a data.Collection whose Load always fails with an internal error
type failingCollection struct {
	data.Collection
}

// Load always fails with an internal error
func (collection failingCollection) Load(_ exp.Expression, _ data.Object, _ ...option.Option) error {
	return derp.Internal("service.failingCollection.Load", "Synthetic failure")
}

// TestNewOwnerFromConfig pins how the bootstrap owner account is built from the configuration,
// including the defaults and the whitespace trimming.
func TestNewOwnerFromConfig(t *testing.T) {

	t.Run("FullyConfigured", func(t *testing.T) {
		owner := newOwnerFromConfig(config.Owner{
			DisplayName:  "Ben Pate",
			Username:     "benpate",
			EmailAddress: "ben@pate.org",
		}, "example.com")

		require.Equal(t, "Ben Pate", owner.DisplayName)
		require.Equal(t, "benpate", owner.Username)
		require.Equal(t, "ben@pate.org", owner.EmailAddress)
		require.True(t, owner.IsOwner)
		require.True(t, owner.IsPublic)
	})

	t.Run("BlankFieldsUseDefaults", func(t *testing.T) {
		owner := newOwnerFromConfig(config.Owner{}, "localhost")

		require.Equal(t, "Demo", owner.DisplayName)
		require.Equal(t, "demo", owner.Username)
		require.Equal(t, "demo@localhost", owner.EmailAddress)
		require.True(t, owner.IsOwner)
		require.True(t, owner.IsPublic)
	})

	t.Run("BlankEmailFallsBackToHostname", func(t *testing.T) {
		// A blank email on a public host still yields a valid, non-empty address so that
		// User.Save (which requires an email) succeeds.
		owner := newOwnerFromConfig(config.Owner{
			DisplayName: "Site Admin",
			Username:    "siteadmin",
		}, "example.com")

		require.Equal(t, "demo@example.com", owner.EmailAddress)
	})

	t.Run("WhitespaceIsTrimmed", func(t *testing.T) {
		// The shipped demo config used "admin " (trailing space); it must be trimmed so
		// the username is usable for sign-in and passes username validation.
		owner := newOwnerFromConfig(config.Owner{
			DisplayName:  "  Demo Admin  ",
			Username:     "admin ",
			EmailAddress: "  demo@example.com ",
		}, "example.com")

		require.Equal(t, "Demo Admin", owner.DisplayName)
		require.Equal(t, "admin", owner.Username)
		require.Equal(t, "demo@example.com", owner.EmailAddress)
	})

	t.Run("WhitespaceOnlyFieldsUseDefaults", func(t *testing.T) {
		owner := newOwnerFromConfig(config.Owner{
			DisplayName:  "   ",
			Username:     "   ",
			EmailAddress: "   ",
		}, "localhost")

		require.Equal(t, "Demo", owner.DisplayName)
		require.Equal(t, "demo", owner.Username)
		require.Equal(t, "demo@localhost", owner.EmailAddress)
	})
}

// TestNeedsHostnameStamp pins when the stored hostname is rewritten from the configuration, which
// wins unless it is blank.
func TestNeedsHostnameStamp(t *testing.T) {

	testCases := []struct {
		name       string
		stored     string
		configured string
		expected   bool
	}{
		{"LegacyRecordWithNoHostname", "", "example.com", true},
		{"RenamedInSetupTool", "old.example.com", "new.example.com", true},
		{"AlreadyMatches", "example.com", "example.com", false},
		{"BlankConfigNeverClearsStored", "example.com", "", false},
		{"BothBlank", "", "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := needsHostnameStamp(testCase.stored, testCase.configured)
			require.Equal(t, testCase.expected, result)
		})
	}
}

// TestCalcOwnerInviteMethod pins how a new owner receives a first password.  A known default
// password is acceptable on localhost only.
func TestCalcOwnerInviteMethod(t *testing.T) {

	testCases := []struct {
		name        string
		isLocalhost bool
		ownerEmail  string
		expected    ownerInviteMethod
	}{
		{"LocalhostWithEmail", true, "ben@pate.org", ownerInviteLocalhost},
		{"LocalhostWithoutEmail", true, "", ownerInviteLocalhost},
		{"PublicWithEmail", false, "ben@pate.org", ownerInviteEmail},
		{"PublicWithoutEmail", false, "", ownerInviteManual},
		{"PublicWithWhitespaceEmail", false, "   ", ownerInviteManual},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := calcOwnerInviteMethod(testCase.isLocalhost, testCase.ownerEmail)
			require.Equal(t, testCase.expected, result)
		})
	}
}

// TestNewOAuthClient_EmptyProviderID pins that a failed Connection load returns an error, rather
// than panicking on the zero Connection's nil Data map.
func TestNewOAuthClient_EmptyProviderID(t *testing.T) {

	domainService := Domain{connectionService: &Connection{}}

	// An empty providerID is rejected by LoadOrCreateByProvider before it touches the session,
	// so a nil session is enough to reach the guard -- and is the proof it returns rather than
	// dereferencing anything.
	connection, err := domainService.NewOAuthClient(nil, "")

	require.Error(t, err)
	require.Equal(t, model.Connection{}, connection)
}

// TestDomain_Get_ZeroValue pins that a Domain service that has never loaded a record returns a
// blank Domain, and the SAME blank Domain on every call.
func TestDomain_Get_ZeroValue(t *testing.T) {

	service := Domain{}

	first := service.Get()
	require.NotNil(t, first)
	require.Equal(t, "default", first.ThemeID)
	require.NotNil(t, first.Connections)
	require.Same(t, first, service.Get())
}

// TestDomain_Publish pins that a published record replaces the cached one
func TestDomain_Publish(t *testing.T) {

	service := NewDomain()

	domain := model.NewWritableDomain()
	domain.Label = "Published"
	service.publish(domain)

	require.Equal(t, "Published", service.Get().Label)
}

// TestDomain_Publish_KeepsOldSnapshots pins that publishing never modifies a record a reader
// already holds, which is what makes a background reload safe.
func TestDomain_Publish_KeepsOldSnapshots(t *testing.T) {

	service := NewDomain()

	before := model.NewWritableDomain()
	before.Label = "Before"
	service.publish(before)

	held := service.Get()

	after := model.NewWritableDomain()
	after.Label = "After"
	service.publish(after)

	require.Equal(t, "Before", held.Label)
	require.Equal(t, "After", service.Get().Label)
	require.NotSame(t, held, service.Get())
}

// TestDomain_Publish_KeepsItsOwnCopy pins that a writer which keeps editing after publish never
// reaches the cache, which is what lets a builder's pipeline continue past its save step.
func TestDomain_Publish_KeepsItsOwnCopy(t *testing.T) {

	service := NewDomain()

	domain := newPopulatedDomain("example.com")
	service.publish(domain)

	domain.Label = "Edited"
	domain.Data["sso_secret"] = "edited"
	domain.Connections["stripe"] = model.Connection{ProviderID: "edited"}
	domain.StartupTasks[0] = "/edited"

	require.Equal(t, newPopulatedDomain("example.com").Domain, *service.Get())
	requireSharesNoMapOrSlice(t, domain.Domain, *service.Get())
}

// TestDomain_Publish_Concurrent pins that readers and a background publisher can run at once.
// It proves nothing without -race.
func TestDomain_Publish_Concurrent(t *testing.T) {

	service := NewDomain()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for index := range 1000 {
			domain := model.NewWritableDomain()
			domain.DatabaseVersion = uint(index)
			service.publish(domain)
		}
	}()

	for range 1000 {
		_ = service.Get().Label
	}

	<-done
	require.Equal(t, uint(999), service.Get().DatabaseVersion)
}

// TestDomain_ObjectLoad pins that ObjectLoad returns a copy of the stored record, whatever the
// criteria, and never the cached one
func TestDomain_ObjectLoad(t *testing.T) {

	t.Run("ReturnsTheStoredRecord", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		storeTestDomain(t, session, "example.com")

		cached := model.NewWritableDomain()
		cached.Label = "Cached"
		service.publish(cached)

		object, err := service.ObjectLoad(session, exp.Equal("_id", primitive.NewObjectID()))
		require.NoError(t, err)

		domain, ok := object.(*model.WritableDomain)
		require.True(t, ok)
		require.Equal(t, "Stored Label", domain.Label)
		require.NotSame(t, service.Get(), &domain.Domain)
	})

	t.Run("NotFoundWhenNothingStored", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")

		_, err := service.ObjectLoad(session, exp.All())
		require.True(t, derp.IsNotFound(err), "expected NotFound, got %v", err)
	})
}

// TestDomain_Load pins that Load reads the stored record into the caller's value, and reports
// NotFound when nothing is stored.
func TestDomain_Load(t *testing.T) {

	t.Run("NotFoundWhenNothingStored", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")

		result := model.NewWritableDomain()
		err := service.Load(session, &result)
		require.True(t, derp.IsNotFound(err), "expected NotFound, got %v", err)
	})

	t.Run("ReadsTheStoredRecord", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		stored := storeTestDomain(t, session, "example.com")

		result := model.NewWritableDomain()
		require.NoError(t, service.Load(session, &result))

		require.Equal(t, "Stored Label", result.Label)
		require.Equal(t, stored.CreateDate, result.CreateDate)
	})

	t.Run("SharesNothingWithTheCache", func(t *testing.T) {

		_, session := newReplicaSetSession(t)
		service := NewDomain()

		populated := newPopulatedDomain("example.com")
		require.NoError(t, service.Save(session, &populated, "test"))

		loaded := model.NewWritableDomain()
		require.NoError(t, service.Load(session, &loaded))

		require.Equal(t, service.Get().Label, loaded.Label)
		requireSharesNoMapOrSlice(t, loaded.Domain, *service.Get())

		// Editing the loaded record, all the way down, leaves the cache untouched
		loaded.Connections["stripe"].Data["mode"] = "edited"
		require.NotContains(t, service.Get().Connections["stripe"].Data, "mode")
	})

	t.Run("ReadsItsOwnWriteInsideATransaction", func(t *testing.T) {

		server, _ := newReplicaSetSession(t)
		service := NewDomain()

		_, err := server.WithTransaction(context.Background(), func(txn data.Session) (any, error) {

			first := model.NewWritableDomain()
			first.Hostname = "example.com"
			first.Label = "First"

			if err := service.Save(txn, &first, "test"); err != nil {
				return nil, err
			}

			second := model.NewWritableDomain()

			if err := service.Load(txn, &second); err != nil {
				return nil, err
			}

			require.Equal(t, "First", second.Label)
			return nil, nil
		})

		require.NoError(t, err)
	})
}

// TestDomain_Save pins that Save publishes exactly what persist wrote, and publishes nothing when
// the write fails.
func TestDomain_Save(t *testing.T) {

	t.Run("PublishesItsOwnCopy", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")

		domain := newPopulatedDomain("example.com")
		require.NoError(t, service.Save(session, &domain, "test"))

		// A writer that keeps editing after Save never reaches the cache
		domain.Data["after"] = "after"
		require.NotContains(t, service.Get().Data, "after")
		requireSharesNoMapOrSlice(t, domain.Domain, *service.Get())
	})

	t.Run("RejectedSaveLeavesTheDatabaseUnchanged", func(t *testing.T) {

		_, session := newReplicaSetSession(t)
		service := NewDomain()

		populated := newPopulatedDomain("example.com")
		require.NoError(t, service.Save(session, &populated, "test"))
		before := service.Get()

		// Edit a loaded record into something the schema refuses
		rejected := model.NewWritableDomain()
		require.NoError(t, service.Load(session, &rejected))
		rejected.Label = "Rejected"
		delete(rejected.Connections, "stripe")
		rejected.ColorMode = "NOT-A-COLOR-MODE"
		require.Error(t, service.Save(session, &rejected, "test"))

		// Neither the database nor the cache saw any of it
		stored := model.NewWritableDomain()
		require.NoError(t, service.Load(session, &stored))
		require.Equal(t, "Populated", stored.Label)
		require.Contains(t, stored.Connections, "stripe")
		require.Same(t, before, service.Get())
	})

	t.Run("PublishesWhatWasWritten", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")

		domain := model.NewWritableDomain()
		domain.Hostname = "example.com"
		domain.Label = "Saved"
		domain.MLSMode = model.DomainMLSModeAll
		domain.MLSGroupIDs = sliceof.String{"group"}
		require.NoError(t, service.Save(session, &domain, "test"))

		// persist clears group IDs outside GROUPS mode, in both copies
		stored := loadStoredDomain(t, session)
		require.Equal(t, "Saved", stored.Label)
		require.Empty(t, stored.MLSGroupIDs)
		require.Equal(t, "Saved", service.Get().Label)
		require.Empty(t, service.Get().MLSGroupIDs)

		// The journal that persist stamped lands on the caller's value and in the cache
		require.NotZero(t, domain.CreateDate)
		require.Equal(t, stored.CreateDate, domain.CreateDate)
		require.Equal(t, stored.CreateDate, service.domain.Load().CreateDate)
	})

	t.Run("KeepsGroupIDsInGroupsMode", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")

		domain := model.NewWritableDomain()
		domain.MLSMode = model.DomainMLSModeGroups
		domain.MLSGroupIDs = sliceof.String{"group"}
		require.NoError(t, service.Save(session, &domain, "test"))

		require.Equal(t, sliceof.String{"group"}, loadStoredDomain(t, session).MLSGroupIDs)
		require.Equal(t, sliceof.String{"group"}, service.Get().MLSGroupIDs)
	})

	t.Run("InvalidRecordIsNotPublished", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		before := service.Get()

		domain := model.NewWritableDomain()
		domain.ColorMode = "NOT-A-COLOR-MODE"
		require.Error(t, service.Save(session, &domain, "test"))

		require.Same(t, before, service.Get())
		requireNothingStored(t, session)
	})

	t.Run("DatabaseErrorIsNotPublished", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		before := service.Get()

		// data-mock refuses any save whose note starts with "ERROR"
		domain := model.NewWritableDomain()
		require.Error(t, service.Save(session, &domain, "ERROR: synthetic"))

		require.Same(t, before, service.Get())
		requireNothingStored(t, session)
	})
}

// TestDomain_Start pins what Start publishes and stores for a new domain, an existing one, and a
// renamed one, and what it leaves behind when it fails.
func TestDomain_Start(t *testing.T) {

	t.Run("FirstRunBootstraps", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		require.NoError(t, service.Start())

		stored := loadStoredDomain(t, session)
		require.Equal(t, "example.com", stored.Hostname)
		require.Equal(t, "Test Domain", stored.Label)
		require.NotZero(t, stored.CreateDate)

		require.Equal(t, "example.com", service.Get().Hostname)
		require.Equal(t, "Test Domain", service.Get().Label)
		require.Equal(t, stored.CreateDate, service.domain.Load().CreateDate)
	})

	t.Run("PublishesTheStoredRecord", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		before := storeTestDomain(t, session, "example.com")

		require.NoError(t, service.Start())
		require.Equal(t, "Stored Label", service.Get().Label)

		// The hostname already matches, so nothing is written back
		require.Equal(t, before.UpdateDate, loadStoredDomain(t, session).UpdateDate)
	})

	t.Run("StampsARenamedHostname", func(t *testing.T) {

		service, session := newTestDomainService(t, "new.example.com")
		storeTestDomain(t, session, "old.example.com")

		require.NoError(t, service.Start())

		stored := loadStoredDomain(t, session)
		require.Equal(t, "new.example.com", stored.Hostname)
		require.Equal(t, "Stored Label", stored.Label)
		require.Equal(t, "new.example.com", service.Get().Hostname)
		require.Equal(t, "Stored Label", service.Get().Label)
	})

	t.Run("BlankHostnameKeepsTheStoredOne", func(t *testing.T) {

		service, session := newTestDomainService(t, "")
		storeTestDomain(t, session, "example.com")

		require.NoError(t, service.Start())
		require.Equal(t, "example.com", loadStoredDomain(t, session).Hostname)
		require.Equal(t, "example.com", service.Get().Hostname)
	})

	t.Run("StampFailureIsReturned", func(t *testing.T) {

		service, session := newTestDomainService(t, "new.example.com")

		// A stored record that fails validation cannot be saved back with its new hostname
		stored := model.NewWritableDomain()
		stored.Hostname = "old.example.com"
		stored.ColorMode = "NOT-A-COLOR-MODE"
		require.NoError(t, session.Collection("Domain").Save(&stored, "Stored"))

		require.Error(t, service.Start())
		require.Equal(t, "old.example.com", loadStoredDomain(t, session).Hostname)
		require.Equal(t, "old.example.com", service.Get().Hostname)
	})

	t.Run("SessionErrorChangesNothing", func(t *testing.T) {

		service, _ := newTestDomainService(t, "example.com")
		service.newSession = func(time.Duration) (data.Session, context.CancelFunc, error) {
			return nil, nil, derp.Internal("test", "Synthetic failure")
		}

		before := service.Get()
		require.Error(t, service.Start())
		require.Same(t, before, service.Get())
	})

	t.Run("LoadErrorLeavesABlankRecord", func(t *testing.T) {

		service, _ := newTestDomainService(t, "example.com")
		service.newSession = func(time.Duration) (data.Session, context.CancelFunc, error) {
			return failingSession{}, func() {}, nil
		}

		published := model.NewWritableDomain()
		published.Label = "Published"
		service.publish(published)

		// Pins current behavior: the reset before the Load stays published when the Load fails
		require.Error(t, service.Start())
		require.Empty(t, service.Get().Label)
		require.Equal(t, "default", service.Get().ThemeID)
	})

	t.Run("TransactionErrorStoresAndPublishesNothing", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		service.withTransaction = func(context.Context, data.TransactionCallbackFunc) (any, error) {
			return nil, derp.Internal("test", "Synthetic failure")
		}

		require.Error(t, service.Start())
		requireNothingStored(t, session)
		require.Empty(t, service.Get().Hostname)
	})

	t.Run("InvalidRecordStoresAndPublishesNothing", func(t *testing.T) {

		service, session := newTestDomainService(t, "example.com")
		service.configuration.Label = strings.Repeat("x", 129) // longer than the schema allows

		require.Error(t, service.Start())
		requireNothingStored(t, session)
		require.Empty(t, service.Get().Hostname)
	})
}

// TestDomain_StampHostname pins the two paths that Start does not reach through a clean record
func TestDomain_StampHostname(t *testing.T) {

	t.Run("NoOpWhenAlreadyStamped", func(t *testing.T) {

		service := NewDomain()
		service.hostname = "example.com"

		domain := model.NewWritableDomain()
		domain.Hostname = "example.com"
		service.publish(domain)

		// A nil session proves that nothing is written
		require.NoError(t, service.stampHostname(nil))
	})

	t.Run("SaveErrorIsReturned", func(t *testing.T) {

		service, session := newTestDomainService(t, "new.example.com")

		// The stored record fails validation, so it cannot be saved back with its new hostname
		stored := model.NewWritableDomain()
		stored.Hostname = "old.example.com"
		stored.ColorMode = "NOT-A-COLOR-MODE"
		require.NoError(t, session.Collection("Domain").Save(&stored, "Stored"))
		service.publish(stored)

		require.Error(t, service.stampHostname(session))
		require.Equal(t, "old.example.com", service.Get().Hostname)
		require.Equal(t, "old.example.com", loadStoredDomain(t, session).Hostname)
	})

	t.Run("LoadErrorIsReturned", func(t *testing.T) {

		service := NewDomain()
		service.hostname = "new.example.com"

		domain := model.NewWritableDomain()
		domain.Hostname = "old.example.com"
		service.publish(domain)

		require.Error(t, service.stampHostname(failingSession{}))
		require.Equal(t, "old.example.com", service.Get().Hostname)
	})

	t.Run("StoredRecordAlreadyStamped", func(t *testing.T) {

		service, session := newTestDomainService(t, "new.example.com")
		before := storeTestDomain(t, session, "new.example.com")

		// The cache is stale, but the stored record already carries the hostname, so nothing is written
		stale := model.NewWritableDomain()
		stale.Hostname = "old.example.com"
		service.publish(stale)

		require.NoError(t, service.stampHostname(session))
		require.Equal(t, before.Revision, loadStoredDomain(t, session).Revision)
	})
}

// TestDomain_Save_ReachesAnotherServer pins BUG-170 end to end: a Domain saved by one server is
// published on another through queries.WatchDomain, with no restart.
func TestDomain_Save_ReachesAnotherServer(t *testing.T) {

	server := mongodb.NewServer(newDomainTestDatabase(t))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Two Domain services stand in for two servers that share one database
	saving := NewDomain()
	watching := NewDomain()
	go queries.WatchDomain(ctx, server, watching.publish)

	session, err := server.Session(ctx)
	require.NoError(t, err)

	// The first save inserts the record, which the watcher reports once its stream is open
	domain := model.NewWritableDomain()
	domain.Hostname = "example.com"
	domain.Label = "First"
	require.NoError(t, saving.Save(session, &domain, "test"))
	require.Eventually(t, func() bool { return watching.Get().Label == "First" }, 10*time.Second, 20*time.Millisecond)

	// The second save replaces the record, and can only arrive as a change event
	updated := model.NewWritableDomain()
	require.NoError(t, saving.Load(session, &updated))
	updated.Label = "Second"
	require.NoError(t, saving.Save(session, &updated, "test"))
	require.Eventually(t, func() bool { return watching.Get().Label == "Second" }, 10*time.Second, 20*time.Millisecond)
}
