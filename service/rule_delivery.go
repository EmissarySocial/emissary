package service

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DeliveryFilter returns a test that tells whether an activity from this User (or from the
// admin tier alone, for NilObjectID) may be delivered to a recipient. It loads the block rules
// once, so that a fan-out can test every recipient without another query.
func (service *Rule) DeliveryFilter(session data.Session, userID primitive.ObjectID) (func(recipientURL string) bool, error) {

	const location = "service.Rule.DeliveryFilter"

	// Only a BLOCK halts delivery. A mute never gates egress (D5).
	criteria := service.byUserID(userID).And(exp.Equal("action", model.RuleActionBlock))
	rules, err := service.QuerySummary(session, criteria)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading block rules", "userId: "+userID.Hex())
	}

	return newDeliveryFilter(rules, time.Now().Unix()), nil
}

// newDeliveryFilter returns a recipient test over a fixed set of block rules
func newDeliveryFilter(rules []model.RuleSummary, now int64) func(recipientURL string) bool {

	// Index the rules by the key that a matching recipient produces
	byMatchKey := make(map[string][]model.RuleSummary, len(rules))

	for _, rule := range rules {
		byMatchKey[rule.MatchKey] = append(byMatchKey[rule.MatchKey], rule)
	}

	return func(recipientURL string) bool {

		keys := deliveryMatchKeys(recipientURL)

		// RULE: A recipient that cannot be identified cannot be cleared, so it is not delivered to
		if len(keys) == 0 {
			return false
		}

		matching := make([]model.RuleSummary, 0)

		for _, key := range keys {
			matching = append(matching, byMatchKey[key]...)
		}

		// One matcher decides here and at the inbox gate, so the two agree on expiry and severity
		return !model.NewRuleDispositionForKeys(keys, matching, now).IsBlocked()
	}
}

// deliveryMatchKeys returns the rule keys for a recipient: the actor and its host's domains for
// an actor URL, and the domains alone for an email address
func deliveryMatchKeys(recipientURL string) []string {

	// RULE: A domain block also stops email to that domain. An email address has no actor key.
	if address, isEmail := strings.CutPrefix(recipientURL, "mailto:"); isEmail {

		_, domain, found := strings.Cut(address, "@")

		if !found || (domain == "") {
			return nil
		}

		return model.DomainMatchKeys("https://" + domain)
	}

	return model.ActorMatchKeys(recipientURL)
}
