// Package service is Emissary's business-logic layer.  Each service -- Stream, User, Follower,
// Following, Conversation, Inbox, Outbox, and many more -- owns every interaction with one model
// collection: loading, saving, validating, and coordinating the side effects of a change.
//
// Services are where the rules live.  Handlers and builders call into services rather than
// touching the database themselves, so a rule has exactly one home and cannot be enforced in one
// caller and forgotten in the next.
//
// Services are assembled and wired together by the Factory, one per domain, which hands each
// service the collaborators it needs.  Data access is passed IN as a data.Session rather than
// held by the service, so that a caller can run a sequence of service calls inside a single
// transaction.
//
// A service typically spans several files by role: a trailing-underscore file (stream_.go) for
// the record lifecycle, and siblings for each concern layered on top of it -- _activitypub for
// federated representation, _publish for outbound delivery, and so on.
//
// Subdirectories hold pluggable integrations: providers for external services such as Stripe
// Connect, and geocoder for address lookups.
package service
