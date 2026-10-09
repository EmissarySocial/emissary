package model

/******************************************
 * User Connection Types
 * the external services a User can connect
 * to on their own behalf
 ******************************************/

// UserConnectionTypeMailchimp identifies a connection to a User's own Mailchimp account
const UserConnectionTypeMailchimp = "MAILCHIMP"

/******************************************
 * User Connection Status
 ******************************************/

// UserConnectionStatusPending marks a connection that is not yet set up at the remote service:
// brand new, switched off, or a proven credential with nowhere to sync to yet
const UserConnectionStatusPending = "PENDING"

// UserConnectionStatusReady marks a connection whose setup at the remote service has completed
const UserConnectionStatusReady = "READY"

// UserConnectionStatusReconnect marks a connection whose credentials the remote service rejected
const UserConnectionStatusReconnect = "RECONNECT"

/******************************************
 * Connection Values
 *
 * Every key a UserConnection stores is named here, so that no call site and no
 * Template spells one by hand. Keys are NOT prefixed by service, because the
 * record already belongs to exactly one.
 ******************************************/

// UserConnectionVaultAPIKey is the Vault key holding the credential this connection authenticates with
const UserConnectionVaultAPIKey = "apiKey" // #nosec G101 -- this is the NAME of a Vault key, not a credential

// UserConnectionVaultWebhookSecret is the Vault key holding the secret that authenticates
// inbound webhook deliveries from the remote service
const UserConnectionVaultWebhookSecret = "webhookSecret" // #nosec G101 -- this is the NAME of a Vault key, not a credential

// UserConnectionDataCenter is the Data key holding the Mailchimp data center that addresses
// this User's account, such as "us6"
const UserConnectionDataCenter = "dataCenter"

// UserConnectionDataAudienceID is the Data key holding the remote audience (list) that this
// User's followers are pushed into
const UserConnectionDataAudienceID = "audienceId"

// UserConnectionDataAudienceName is the Data key holding that audience's human-readable
// name, echoed back to the User so a pasted ID can be seen to be the right one
const UserConnectionDataAudienceName = "audienceName"

// UserConnectionDataWebhookID is the Data key holding the ID of the webhook Emissary installed,
// so that disconnecting can remove it
const UserConnectionDataWebhookID = "webhookId" // #nosec G101 -- this is the NAME of a Data key, not a credential

// UserConnectionDataTag is the Data key holding the optional Mailchimp tag that is applied to
// every member Emissary pushes into the audience
const UserConnectionDataTag = "tag"
