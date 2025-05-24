package model

/******************************************
 * Connection Types
 * defines that specific that a provider
 * performs.
 ******************************************/

// ConnectionTypeGeocoder represents a provider that can geocode addresses
const ConnectionTypeGeocoder = "GEOCODER"

// ConnectionTypeImage represents a provider/connection that can be used to generate images
const ConnectionTypeImage = "IMAGE"

// ConnectionTypeUserUserPayment represents a provider that can take payments for users
const ConnectionTypeUserPayment = "USER-PAYMENT"

/******************************************
 * Provider Definitions
 * definitions of the services that have
 * been implemented in the system.
 ******************************************/

// ConnectionProviderArcGIS represents an API connection to the https://www.arcgis.com service
const ConnectionProviderArcGIS = "ARCGIS"

const ConnectionProviderBing = "BING"

// ConnectionProviderGoogleMaps represents an API connection to the https://maps.google.com service
const ConnectionProviderGoogleMaps = "GOOGLE-MAPS"

// ConnectionProviderOpenStreetMap represents an API connection to the https://openstreetmap.org service
const ConnectionProviderOpenStreetMap = "OPENSTREETMAP"

const ConnectionProviderTomTom = "TOMTOM"

// ConnectionProviderGiphy represents an API connection to the https://giphy.com service
// for generating animated GIFs.
const ConnectionProviderGiphy = "GIPHY"

// ConnectionProviderPayPal represents an API connection to the https://paypal.com service
// for processing payments.
const ConnectionProviderPayPal = "PAYPAL"

// ConnectionProviderStripe represents an API connection to the https://stripe.com service
// for processing payments.
const ConnectionProviderStripe = "STRIPE"

// ConnectionProviderUnsplash represents an API connection to the https://unsplash.com service
// for generating photographs.
const ConnectionProviderUnsplash = "UNSPLASH"
