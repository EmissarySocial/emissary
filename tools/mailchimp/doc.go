// Package mailchimp is a thin client for the Mailchimp Marketing API v3.
//
// It speaks HTTP and knows nothing about Emissary's services, sessions, or model
// objects, so every function here can be exercised against an httptest server with
// no Factory and no database. Callers pass a credential in and get typed values back.
//
// Mailchimp publishes no official Go SDK, so this package is hand-rolled over
// benpate/remote.
//
// Two rules shape the package. A credential is OPAQUE: nothing here parses one, and
// only Mailchimp decides whether one works. The data center that routes a request is
// therefore a separate value the User supplies, and BaseURL is the only thing in
// Emissary that turns it into an address. README.md explains both.
package mailchimp
