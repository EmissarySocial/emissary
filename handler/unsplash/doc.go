// Package unsplash proxies image lookups against the Unsplash API.
//
// GetPhoto returns a single photo and GetCollectionRandom picks one at random from a
// collection, both rendering the attribution block that the Unsplash API terms require
// alongside every displayed image.
//
// Requests are proxied rather than made from the browser so the API key stays on the server,
// and so a visitor's address is never handed to Unsplash.
package unsplash
