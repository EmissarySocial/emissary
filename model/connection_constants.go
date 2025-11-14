package model

/******************************************
 * Connection Types
 * defines that specific roles that a
 * provider fulfills.
 ******************************************/

// ConnectionTypeGeocodeAddress represents a connection that geocodes individual physical addresses (and often place names)
const ConnectionTypeGeocodeAddress = "GEOCODE-ADDRESS"

// ConnectionTypeGeocodeAutocomplete represents a connection that searches addresses / place names
const ConnectionTypeGeocodeAutocomplete = "GEOCODE-AUTOCOMPLETE"

// ConnectionTypeGeocodeNetwork represents a connection that geocodes individual IP addresses
const ConnectionTypeGeocodeNetwork = "GEOCODER-NETWORK"

// ConnectionTypeGeocodeTiles represents an API connection to a map tile provider
const ConnectionTypeGeocodeTiles = "GEOCODE-TILES"

// ConnectionTypeGeocodeTimezone represents an API connection to a timezone provider
const ConnectionTypeGeocodeTimezone = "GEOCODE-TIMEZONE"

// ConnectionTypeImage represents a connection that can be used to generate images
const ConnectionTypeImage = "IMAGE"

// ConnectionTypeUserUserPayment represents a connection that can take payments for users
const ConnectionTypeUserPayment = "USER-PAYMENT"

/******************************************
 * Provider Definitions
 * definitions of the services that have
 * been implemented in the system.
 ******************************************/

// ConnectionProviderGeocodeAddress represents an API connection to a physical address geocoder
const ConnectionProviderGeocodeAddress = "GEOCODE-ADDRESS"

// ConnectionProviderGeocodeAutocomplete represents and API connection to a geocode autocomplete provider
const ConnectionProviderGeocodeAutocomplete = "GEOCODE-AUTOCOMPLETE"

// ConnectionProviderGeocodeNetwork represents a connection to an IP address geocoder.
const ConnectionProviderGeocodeNetwork = "GEOCODE-NETWORK"

// ConnectionProviderGeocodeTimezone represents a connection to a timezone geocoder.
const ConnectionProviderGeocodeTimezone = "GEOCODE-TIMEZONE"

// ConnectionProviderGeocodeTiles represents a connection to a map tile provider.
const ConnectionProviderGeocodeTiles = "GEOCODE-TILES"

// ConnectionProviderGiphy represents an API connection to the https://giphy.com service
// for generating animated GIFs.
const ConnectionProviderGiphy = "GIPHY"

// ConnectionProviderStripe represents an API connection to the https://stripe.com service
// for processing payments, using direct API keys.
const ConnectionProviderStripe = "STRIPE"

// ConnectionProviderStripeConnect represents an API connection to the https://stripe.com service
// for processing payments, using the Stripe Connect / OAuth authentication.
const ConnectionProviderStripeConnect = "STRIPE-CONNECT"

// ConnectionProviderUnsplash represents an API connection to the https://unsplash.com service
// for generating photographs.
const ConnectionProviderUnsplash = "UNSPLASH"
