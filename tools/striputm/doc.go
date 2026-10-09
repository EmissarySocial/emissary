// Package striputm removes campaign-tracking parameters from a URL.
//
// StripFromURL deletes the known tracking codes -- the utm_* family and the per-network
// click identifiers that behave the same way -- from a URL's query string, in place.
// KnownCodes reports the list it removes.
//
// Emissary normalizes links this way before storing or comparing them, so that the same
// article shared through two campaigns is recognized as one document rather than two, and so
// a reader's referral trail is not republished to everyone who sees the link.
package striputm
