// Package geocoder resolves places, addresses, and network addresses into coordinates.
//
// Emissary supports several geocoding vendors, and each one gets a type here that wraps its
// HTTP API.  They implement the narrow interfaces the rest of the application depends on --
// AddressGeocoder, NetworkGeocoder, and their siblings -- so a Domain can be configured with
// whichever vendor its operator has an account for, and the calling code does not change.
//
// Every vendor is reached with an API key supplied by configuration, and every one returns a
// best guess: a lookup that finds nothing is an empty result, not an error.
package geocoder
