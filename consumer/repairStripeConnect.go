package consumer

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// RepairStripeConnect replaces a Domain's Stripe Connect webhook endpoint whose signing secret was
// never stored, then queues a reconciliation of the Privileges that its failed webhooks missed.
func RepairStripeConnect(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.RepairStripeConnect"

	repaired, err := factory.Connection().RepairStripeConnect(session)

	// A refusal from Stripe, such as a revoked key, fails once and is tried again at the next boot
	if err != nil {
		return requeue(derp.Wrap(err, location, "Repairing Stripe Connect webhook", factory.Hostname()))
	}

	// Another server already repaired it, and queued the reconciliation itself
	if !repaired {
		return queue.Success()
	}

	log.Warn().
		Str("loc", location).
		Str("hostname", factory.Hostname()).
		Msg("Replaced a Stripe Connect webhook endpoint whose signing secret was never stored")

	// Reconcile in a task of its own, so that a failed Stripe lookup retries without repairing again
	postcommit.Publish(
		session,
		factory.Queue(),
		"ReconcileStripeSubscriptions",
		mapof.Any{"hostname": factory.Hostname()},
		queue.WithSignature("ReconcileStripeSubscriptions:"+factory.Hostname()),
	)

	// Good as new
	return queue.Success()
}

// ReconcileStripeSubscriptions revokes every Stripe Connect Privilege on a Domain whose subscription
// has ended, which its webhooks would have done had they been verified.
func ReconcileStripeSubscriptions(factory *service.Factory, _ mapof.Any) queue.Result {

	const location = "consumer.ReconcileStripeSubscriptions"

	// RULE: No transaction.  Each revocation stands alone, and one Stripe call per subscription
	// would soon outlast MongoDB's 60-second limit on a transaction.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	session, err := factory.Server().Session(ctx)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Opening database session", factory.Hostname()))
	}

	defer session.Close()

	revoked, err := factory.MerchantAccount().ReconcileStripeSubscriptions(session)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Reconciling Stripe subscriptions", factory.Hostname()))
	}

	log.Warn().
		Str("loc", location).
		Str("hostname", factory.Hostname()).
		Int("revoked", revoked).
		Msg("Reconciled Stripe Connect subscriptions")

	// The ledger agrees
	return queue.Success()
}
