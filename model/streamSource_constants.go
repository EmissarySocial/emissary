package model

// StreamSourceConfigWebhookToken is the Config key that holds the token in a record's webhook URL.
// The value is deliberately NOT unique: many records may share one token, so that one ping from a
// repository refreshes every page sourced from it.
const StreamSourceConfigWebhookToken = "webhookToken"

// StreamSourceWebhookTokenMinLength is the shortest webhook token that may be saved or accepted.
// It is enforced on save AND at the endpoint, because the value is editable by hand.
const StreamSourceWebhookTokenMinLength = 16

// webhookTokenNonceBytes is the amount of randomness behind a generated webhook token
const webhookTokenNonceBytes = 32

// StreamSourceMethodHTTPS identifies a StreamSource record that reads a Markdown file from a URL
const StreamSourceMethodHTTPS = "HTTPS"

// StreamSourceStatusNew marks a StreamSource record that has never been synchronized
const StreamSourceStatusNew = "NEW"

// StreamSourceStatusLoading marks a StreamSource record whose synchronization is under way
const StreamSourceStatusLoading = "LOADING"

// StreamSourceStatusSuccess marks a StreamSource record whose last synchronization succeeded
const StreamSourceStatusSuccess = "SUCCESS"

// StreamSourceStatusFailure marks a StreamSource record whose last synchronization failed
const StreamSourceStatusFailure = "FAILURE"
