package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v78"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stripeSubscriptionLoader retrieves one Stripe subscription.  stripeapi.Subscription is the only
// production implementation; tests supply their own.
type stripeSubscriptionLoader func(restrictedKey string, connectedAccountID string, subscriptionID string) (stripe.Subscription, error)

// ReconcileStripeSubscriptions revokes every Stripe Connect Privilege whose subscription is no longer
// active, and returns how many it revoked.
func (service *MerchantAccount) ReconcileStripeSubscriptions(session data.Session) (int, error) {
	return service.reconcileStripeSubscriptions(session, stripeapi.Subscription)
}

// reconcileStripeSubscriptions is ReconcileStripeSubscriptions, with the Stripe lookup supplied
func (service *MerchantAccount) reconcileStripeSubscriptions(session data.Session, loadSubscription stripeSubscriptionLoader) (int, error) {

	const location = "service.MerchantAccount.reconcileStripeSubscriptions"

	// Find every Privilege bought as a subscription.  A one-time purchase stores its checkout
	// session ID ("cs_") instead, and has nothing to reconcile.
	privileges, err := service.privilegeService.Range(session, exp.BeginsWith("remotePurchaseId", "sub_"))

	if err != nil {
		return 0, derp.Wrap(err, location, "Querying subscription Privileges")
	}

	merchantAccounts := make(map[primitive.ObjectID]model.MerchantAccount)
	revoked := 0

	for privilege := range privileges {

		// RULE: A Privilege that no MerchantAccount sold has nothing to reconcile
		if privilege.MerchantAccountID.IsZero() {
			continue
		}

		// Find the MerchantAccount that sold it
		merchantAccount := service.cachedMerchantAccount(session, merchantAccounts, privilege.MerchantAccountID)

		// RULE: Only Stripe Connect subscriptions lost their webhooks
		if merchantAccount.Type != model.ConnectionProviderStripeConnect {
			continue
		}

		// Ask Stripe whether the subscription is still active
		restrictedKey, err := service.stripe_getRestrictedKey(&merchantAccount)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Retrieving restricted key", privilege.PrivilegeID.Hex()))
			continue
		}

		connectedAccountID := service.stripe_getConnectedAccountID(&merchantAccount)
		subscription, err := loadSubscription(restrictedKey, connectedAccountID, privilege.RemotePurchaseID)

		// RULE: Never revoke on an uncertain answer.  A failed lookup is reported and left alone.
		if err != nil {
			derp.Report(derp.Wrap(err, location, "Loading Stripe subscription", privilege.PrivilegeID.Hex()))
			continue
		}

		if stripeapi.SubscriptionIsActive(subscription) {
			continue
		}

		// Revoke it, as the webhook would have.  The note stays on the deleted record.
		status := string(subscription.Status)

		if err := service.privilegeService.Delete(session, &privilege, "Revoked: Stripe subscription is "+status); err != nil {
			return revoked, derp.Wrap(err, location, "Revoking Privilege", privilege.PrivilegeID.Hex())
		}

		log.Warn().
			Str("loc", location).
			Str("privilegeId", privilege.PrivilegeID.Hex()).
			Str("subscriptionStatus", status).
			Msg("Revoked a Privilege whose Stripe subscription has ended")

		revoked++
	}

	// Books balanced
	return revoked, nil
}

// cachedMerchantAccount returns the MerchantAccount with this ID, loading it only once.  One that
// fails to load is reported once, and cached empty so that its Privileges are skipped.
func (service *MerchantAccount) cachedMerchantAccount(session data.Session, cache map[primitive.ObjectID]model.MerchantAccount, merchantAccountID primitive.ObjectID) model.MerchantAccount {

	const location = "service.MerchantAccount.cachedMerchantAccount"

	if merchantAccount, isCached := cache[merchantAccountID]; isCached {
		return merchantAccount
	}

	merchantAccount := model.NewMerchantAccount()

	if err := service.LoadByID(session, merchantAccountID, &merchantAccount); err != nil {
		derp.Report(derp.Wrap(err, location, "Loading MerchantAccount", merchantAccountID.Hex()))
		merchantAccount = model.MerchantAccount{}
	}

	cache[merchantAccountID] = merchantAccount
	return merchantAccount
}
