package service

import (
	"crypto/subtle"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
)

/******************************************
 * Mailchimp Inbound Webhook
 *
 * The public half of the Mailchimp integration: a delivery arrives with no
 * session and no signed-in User, so the connection's own secret is what
 * authorizes it. Every value in the payload is attacker-supplied, and every
 * lookup is scoped to the connection's owner. See MAILING-LISTS.md 1.3.
 ******************************************/

// Mailchimp webhook event names that Emissary listens for (D11)
const (
	mailchimpEventUnsubscribe = "unsubscribe"
	mailchimpEventUpdateEmail = "upemail"
)

// VerifyWebhookSecret returns TRUE if the presented value matches this connection's
// stored webhook secret
func (service *UserConnection) VerifyWebhookSecret(userConnection *model.UserConnection, presented string) bool {

	const location = "service.UserConnection.VerifyWebhookSecret"

	// RULE: a connection that is switched off, or whose credentials were rejected, does not
	// authorize anything -- the webhook it installed may not even be ours any more.
	if !userConnection.IsReady() {
		return false
	}

	// RULE: an empty presented value can never match. This guard and the stored-value one
	// below are deliberately redundant -- either alone is sufficient, and removing BOTH makes
	// ConstantTimeCompare("", "") authorize every caller against an unminted connection.
	if presented == "" {
		return false
	}

	vault, err := service.DecryptVault(userConnection, model.UserConnectionVaultWebhookSecret)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to read webhook secret", userConnection.UserConnectionID))
		return false
	}

	stored := vault.GetString(model.UserConnectionVaultWebhookSecret)

	if stored == "" {
		return false
	}

	// Constant-time, so that a wrong guess reveals nothing about how wrong it was
	return subtle.ConstantTimeCompare([]byte(stored), []byte(presented)) == 1
}

// Mailchimp_ReceiveWebhook applies one inbound Mailchimp event to this connection's Followers
func (service *UserConnection) Mailchimp_ReceiveWebhook(session data.Session, userConnection *model.UserConnection, event string, values mapof.String) error {

	switch event {

	case mailchimpEventUnsubscribe:
		return service.mailchimp_unsubscribe(session, userConnection, values)

	case mailchimpEventUpdateEmail:
		return service.mailchimp_updateEmail(session, userConnection, values)
	}

	// RULE: an event we did not register for is not an error. Mailchimp sends what it sends,
	// and a delivery we have no use for is discarded rather than failed (D11).
	return nil
}

// mailchimp_unsubscribe removes the Follower whose address unsubscribed at Mailchimp
func (service *UserConnection) mailchimp_unsubscribe(session data.Session, userConnection *model.UserConnection, values mapof.String) error {

	const location = "service.UserConnection.mailchimp_unsubscribe"

	follower, err := service.mailchimp_follower(session, userConnection, values.GetString("data[email]"))

	if err != nil {
		return derp.Wrap(err, location, "Loading Follower")
	}

	// A payload naming somebody who does not follow this User is not an error worth
	// surfacing -- it is what a forged request, or a stale delivery, looks like.
	if follower == nil {
		return nil
	}

	// RULE: this deletion must NOT push an unsubscribe back to Mailchimp. The outbound hook
	// in MAILING-LISTS.md 1.2 does not exist yet; when it lands it must skip this path, or
	// every inbound unsubscribe echoes straight back out at the account that sent it.
	if err := service.followerService.Delete(session, follower, "Unsubscribed at Mailchimp"); err != nil {
		return derp.Wrap(err, location, "Deleting Follower", follower.FollowerID)
	}

	return nil
}

// mailchimp_updateEmail moves a Follower to the address they changed it to at Mailchimp
func (service *UserConnection) mailchimp_updateEmail(session data.Session, userConnection *model.UserConnection, values mapof.String) error {

	const location = "service.UserConnection.mailchimp_updateEmail"

	// RULE: the NEW address is as attacker-supplied as the old one, so it is validated
	// before it is written. An empty or malformed value would silently orphan the record.
	newEmail := strings.TrimSpace(values.GetString("data[new_email]"))

	if newEmail == "" {
		return nil
	}

	follower, err := service.mailchimp_follower(session, userConnection, values.GetString("data[old_email]"))

	if err != nil {
		return derp.Wrap(err, location, "Loading Follower")
	}

	if follower == nil {
		return nil
	}

	follower.Actor.EmailAddress = newEmail
	follower.Actor.ProfileURL = newEmail

	if err := service.followerService.Save(session, follower, "Email address changed at Mailchimp"); err != nil {
		return derp.Wrap(err, location, "Saving Follower", follower.FollowerID)
	}

	return nil
}

// mailchimp_follower resolves the Follower named by a webhook payload, or NIL when the
// payload names nobody this connection's owner follows
func (service *UserConnection) mailchimp_follower(session data.Session, userConnection *model.UserConnection, emailAddress string) (*model.Follower, error) {

	const location = "service.UserConnection.mailchimp_follower"

	emailAddress = strings.TrimSpace(emailAddress)

	if emailAddress == "" {
		return nil, nil
	}

	// RULE: scoped to the connection's OWNER, always. The address is attacker-supplied, and
	// an unscoped lookup would let one leaked secret reach every Follower on the server (D27).
	follower := model.NewFollower()

	if err := service.followerService.LoadByEmailAddress(session, userConnection.UserID, emailAddress, &follower); err != nil {

		if derp.IsNotFound(err) {

			// Emissary stores the address it sent to Mailchimp, so the echo should match it
			// exactly. A miss is therefore worth a log line: it means the two have drifted,
			// and D9's "unsubscribe means unsubscribe" is quietly not happening.
			log.Debug().
				Str("userConnectionId", userConnection.UserConnectionID.Hex()).
				Msg("Mailchimp webhook named an address with no matching Follower")

			return nil, nil
		}

		return nil, derp.Wrap(err, location, "Loading Follower by email address")
	}

	return &follower, nil
}
