package service

import (
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v78"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Stripe Subscription Reconciliation
 *
 * The Stripe Connect webhook is the only thing that revokes a Privilege
 * when its subscription ends, so every subscription that ended while the
 * webhooks were failing (BUG-179) is reconciled against Stripe directly.
 ******************************************/

// reconcileReportRecorder is a derp reporter that keeps every error reported to it
type reconcileReportRecorder struct {
	lock   sync.Mutex
	errors []error
}

// Report records one error
func (recorder *reconcileReportRecorder) Report(err error) {
	recorder.lock.Lock()
	defer recorder.lock.Unlock()
	recorder.errors = append(recorder.errors, err)
}

// reconcileFixture holds the services and records that one reconciliation test runs against
type reconcileFixture struct {
	merchantAccountService *MerchantAccount
	session                data.Session
	identityID             primitive.ObjectID
	stripeConnectID        primitive.ObjectID
	directStripeID         primitive.ObjectID
}

// newReconcileFixture stores an Identity, a Stripe Connect MerchantAccount, and a direct Stripe one in a
// throwaway replica-set database, and wires the services that a reconciliation reaches.  It runs
// against MongoDB, because the in-memory database cannot match a field whose tag carries options.
func newReconcileFixture(t *testing.T) reconcileFixture {

	t.Helper()

	_, session := newReplicaSetSession(t)

	identityService := Identity{}
	privilegeService := Privilege{identityService: &identityService}
	identityService.privilegeService = &privilegeService

	merchantAccountService := MerchantAccount{
		privilegeService: &privilegeService,
		encryptionKey:    testDomainCipher,
	}

	// The Identity that every Privilege belongs to, which Privilege.Delete refreshes
	identity := model.NewIdentity()
	require.NoError(t, session.Collection("Identity").Save(&identity, "Stored for test"))

	// A Stripe Connect MerchantAccount, whose key and connected account are sent to Stripe
	stripeConnect := model.NewMerchantAccount()
	stripeConnect.Type = model.ConnectionProviderStripeConnect
	stripeConnect.Vault = newSealedVault(t, "restrictedKey", "rk_test_1")
	stripeConnect.Plaintext.SetString("accountId", "acct_1")
	require.NoError(t, session.Collection("MerchantAccount").Save(&stripeConnect, "Stored for test"))

	// A direct Stripe MerchantAccount, whose webhooks were never broken.  The provider is disabled,
	// so its constant is commented out in model and the stored value is written by hand.
	directStripe := model.NewMerchantAccount()
	directStripe.Type = "STRIPE"
	require.NoError(t, session.Collection("MerchantAccount").Save(&directStripe, "Stored for test"))

	return reconcileFixture{
		merchantAccountService: &merchantAccountService,
		session:                session,
		identityID:             identity.IdentityID,
		stripeConnectID:        stripeConnect.MerchantAccountID,
		directStripeID:         directStripe.MerchantAccountID,
	}
}

// storePrivilege stores a Privilege sold by merchantAccountID under remotePurchaseID, and returns its ID
func (fixture reconcileFixture) storePrivilege(t *testing.T, merchantAccountID primitive.ObjectID, remotePurchaseID string) primitive.ObjectID {

	t.Helper()

	privilege := model.NewPrivilege()
	privilege.IdentityID = fixture.identityID
	privilege.MerchantAccountID = merchantAccountID
	privilege.RemotePurchaseID = remotePurchaseID

	require.NoError(t, fixture.session.Collection("Privilege").Save(&privilege, "Stored for test"))
	return privilege.PrivilegeID
}

// loadPrivilege returns a Privilege as stored, including one that has been deleted
func (fixture reconcileFixture) loadPrivilege(t *testing.T, privilegeID primitive.ObjectID) model.Privilege {

	t.Helper()

	privilege := model.NewPrivilege()
	require.NoError(t, fixture.session.Collection("Privilege").Load(exp.Equal("_id", privilegeID), &privilege))
	return privilege
}

// TestMerchantAccount_ReconcileStripeSubscriptions pins which Privileges are revoked, which are kept,
// and which are never looked up at all.
func TestMerchantAccount_ReconcileStripeSubscriptions(t *testing.T) {

	reports := &reconcileReportRecorder{}
	derp.SetPlugins(reports)
	t.Cleanup(func() { derp.SetPlugins() })

	fixture := newReconcileFixture(t)
	missingMerchantID := primitive.NewObjectID()

	active := fixture.storePrivilege(t, fixture.stripeConnectID, "sub_active")
	canceled := fixture.storePrivilege(t, fixture.stripeConnectID, "sub_canceled")
	trialing := fixture.storePrivilege(t, fixture.stripeConnectID, "sub_trialing")
	lookupFails := fixture.storePrivilege(t, fixture.stripeConnectID, "sub_unknown")
	oneTime := fixture.storePrivilege(t, fixture.stripeConnectID, "cs_onetime")
	direct := fixture.storePrivilege(t, fixture.directStripeID, "sub_direct")
	noMerchant := fixture.storePrivilege(t, primitive.NilObjectID, "sub_nomerchant")
	missingOne := fixture.storePrivilege(t, missingMerchantID, "sub_missing_one")
	missingTwo := fixture.storePrivilege(t, missingMerchantID, "sub_missing_two")

	// A fake Stripe that knows three subscriptions, and records every lookup
	statuses := map[string]stripe.SubscriptionStatus{
		"sub_active":   stripe.SubscriptionStatusActive,
		"sub_canceled": stripe.SubscriptionStatusCanceled,
		"sub_trialing": stripe.SubscriptionStatusTrialing,
	}

	lookups := make([]string, 0)

	loadSubscription := func(restrictedKey string, connectedAccountID string, subscriptionID string) (stripe.Subscription, error) {

		lookups = append(lookups, subscriptionID)

		if (restrictedKey != "rk_test_1") || (connectedAccountID != "acct_1") {
			return stripe.Subscription{}, derp.Unauthorized("fakeStripe", "Wrong key or account", restrictedKey, connectedAccountID)
		}

		status, exists := statuses[subscriptionID]

		if !exists {
			return stripe.Subscription{}, derp.NotFound("fakeStripe", "No such subscription", subscriptionID)
		}

		return stripe.Subscription{ID: subscriptionID, Status: status}, nil
	}

	revoked, err := fixture.merchantAccountService.reconcileStripeSubscriptions(fixture.session, loadSubscription)
	require.NoError(t, err)
	require.Equal(t, 1, revoked)

	// Only Stripe Connect subscriptions are looked up, with that merchant's own key and account
	require.ElementsMatch(t, []string{"sub_active", "sub_canceled", "sub_trialing", "sub_unknown"}, lookups)

	// The ended subscription is revoked, and says why on the deleted record
	revokedPrivilege := fixture.loadPrivilege(t, canceled)
	require.True(t, revokedPrivilege.IsDeleted())
	require.Equal(t, "Revoked: Stripe subscription is canceled", revokedPrivilege.Note)

	// Everything else is kept, including the one Stripe could not answer for
	for _, privilegeID := range []primitive.ObjectID{active, trialing, lookupFails, oneTime, direct, noMerchant, missingOne, missingTwo} {
		require.False(t, fixture.loadPrivilege(t, privilegeID).IsDeleted(), privilegeID.Hex())
	}

	// One report for the failed lookup, and ONE for the MerchantAccount that would not load
	require.Len(t, reports.errors, 2)

	// A second pass finds nothing left to revoke
	lookups = lookups[:0]
	revoked, err = fixture.merchantAccountService.reconcileStripeSubscriptions(fixture.session, loadSubscription)
	require.NoError(t, err)
	require.Zero(t, revoked)
	require.NotContains(t, lookups, "sub_canceled", "a revoked Privilege is not looked up again")
}
