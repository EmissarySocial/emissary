// Package providers describes the external services a Domain or User can connect to.
//
// A provider is the metadata and behavior around one integration: the form its credentials
// are collected with, what to do when a connection is activated or saved, and the adapter
// that other services call once it is live.  Bluesky bridging, the geocoding vendors, Giphy,
// Stripe Connect, and Unsplash each have one here.
//
// Providers hold no credentials themselves.  Secrets live in the Connection or
// UserConnection vault, encrypted with the Domain's master key, and are decrypted only at
// the moment of use.
package providers
