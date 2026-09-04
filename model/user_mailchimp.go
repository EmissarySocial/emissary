package model

/******************************************
 * Mailchimp Connection
 *
 * A User's Mailchimp connection lives in namespaced keys on User.Data and
 * User.Vault, not in a record of its own. Secrets go in the Vault, whose
 * `json:"-"` tags keep them out of the User's data export; remote IDs and
 * routing go in Data. A credential is opaque -- nothing parses one -- so
 * the data center is its own configured value. See MAILING-LISTS.md.
 ******************************************/

// UserMailchimpPrefix is the namespace shared by every Mailchimp key in Data and
// Vault, and the prefix that disconnecting clears by
const UserMailchimpPrefix = "MAILCHIMP-"

// UserVaultMailchimpAPIKey is the Vault key holding the User's Mailchimp API key,
// which carries full account access and no scopes
const UserVaultMailchimpAPIKey = "MAILCHIMP-API" // #nosec G101 -- this is the NAME of a Vault key, not a credential

// UserVaultMailchimpWebhookSecret is the Vault key holding the secret that
// authenticates inbound webhook deliveries from Mailchimp
const UserVaultMailchimpWebhookSecret = "MAILCHIMP-WEBHOOK-SECRET" // #nosec G101 -- this is the NAME of a Vault key, not a credential

// UserDataMailchimpDataCenter is the Data key holding the Mailchimp data center
// that addresses this User's account, such as "us6"
const UserDataMailchimpDataCenter = "MAILCHIMP-DATACENTER"

// UserDataMailchimpAudience is the Data key holding the Mailchimp audience (list) ID
// that this User's followers are pushed into
const UserDataMailchimpAudience = "MAILCHIMP-AUDIENCE"

// UserDataMailchimpTag is the Data key holding the ID of the `EMISSARY` segment
// applied to every member Emissary adds
const UserDataMailchimpTag = "MAILCHIMP-TAG"

// UserDataMailchimpWebhook is the Data key holding the ID of the webhook Emissary
// installed, so that disconnecting can remove it
const UserDataMailchimpWebhook = "MAILCHIMP-WEBHOOK" // #nosec G101 -- this is the NAME of a Data key, not a credential

// UserDataMailchimpState is the Data key holding the connection state, which is one
// of the UserMailchimpState* constants
const UserDataMailchimpState = "MAILCHIMP-STATE"

// UserMailchimpStateActive marks a connection that completed setup and whose
// credentials worked the last time Emissary used them
const UserMailchimpStateActive = "ACTIVE"

// UserMailchimpStateReconnect marks a connection whose API key Mailchimp rejected
// with a 401
const UserMailchimpStateReconnect = "RECONNECT"

/******************************************
 * Mailchimp Accessors
 ******************************************/

// MailchimpIsConfigured returns TRUE if this User has a complete Mailchimp
// connection: both secrets and all three remote IDs
func (user User) MailchimpIsConfigured() bool {

	// RULE: setup is all-or-nothing, and this is the one place that decides so. A push
	// to an audience with no EMISSARY tag cannot be told apart from the User's own
	// lists. HasString checks presence without decrypting, so this needs no master key.
	if !user.Vault.HasString(UserVaultMailchimpAPIKey) {
		return false
	}

	if !user.Vault.HasString(UserVaultMailchimpWebhookSecret) {
		return false
	}

	if user.Data.GetString(UserDataMailchimpDataCenter) == "" {
		return false
	}

	if user.Data.GetString(UserDataMailchimpAudience) == "" {
		return false
	}

	if user.Data.GetString(UserDataMailchimpTag) == "" {
		return false
	}

	if user.Data.GetString(UserDataMailchimpWebhook) == "" {
		return false
	}

	return true
}

// MailchimpIsActive returns TRUE if this User's Mailchimp connection is complete
// and its credentials worked the last time Emissary used them
func (user User) MailchimpIsActive() bool {

	// RULE: this, not MailchimpIsConfigured, is what the sync and the webhook consult.
	// RECONNECT means Mailchimp refuses the key, so nothing it installed still speaks
	// for this User.
	if !user.MailchimpIsConfigured() {
		return false
	}

	return user.MailchimpState() == UserMailchimpStateActive
}

// MailchimpState returns this User's Mailchimp connection state, or an empty
// string when no connection has been set up
func (user User) MailchimpState() string {
	return user.Data.GetString(UserDataMailchimpState)
}

// MailchimpDataCenter returns the Mailchimp data center that addresses this User's
// account, or an empty string when none has been configured
func (user User) MailchimpDataCenter() string {
	return user.Data.GetString(UserDataMailchimpDataCenter)
}

// MailchimpAudienceID returns the Mailchimp audience this User's followers are
// pushed into, or an empty string when none has been chosen
func (user User) MailchimpAudienceID() string {
	return user.Data.GetString(UserDataMailchimpAudience)
}

// MailchimpTagID returns the ID of the `EMISSARY` segment applied to members
// Emissary adds, or an empty string when it has not been created
func (user User) MailchimpTagID() string {
	return user.Data.GetString(UserDataMailchimpTag)
}

// MailchimpWebhookID returns the ID of the webhook Emissary installed on this
// User's audience, or an empty string when none has been installed
func (user User) MailchimpWebhookID() string {
	return user.Data.GetString(UserDataMailchimpWebhook)
}
