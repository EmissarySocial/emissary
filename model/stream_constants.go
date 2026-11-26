package model

// Sharing status tokens returned by Stream.SharingStatus
const (
	SharingStatusPublic        = "PUBLIC"
	SharingStatusAuthenticated = "AUTHENTICATED"
	SharingStatusCircles       = "CIRCLES"
	SharingStatusOwners        = "OWNERS"
)

// Reply policies stored in Stream.ReplyPolicy.  An empty policy means "everyone", so every
// Stream saved before the field existed keeps accepting replies.
const (
	ReplyPolicyEveryone = "everyone"
	ReplyPolicyNobody   = "nobody"
)

// StreamFingerprintPending is the fingerprint of a Stream that followers received before
// fingerprints were recorded.  It is no hash, so it never matches what a Stream would send.
const StreamFingerprintPending = "PENDING"
