package model

/******************************************
 * Connection Types
 * defines that specific roles that a
 * provider fulfills.
 ******************************************/

// ConnectionTypeGeocoder represents a connection that geocodes individual physical addresses (and often place names)
const ConnectionTypeGeocoder = "GEOCODER"

// ConnectionTypeGeocoderID represents a connection that geocodes individual IP addresses
const ConnectionTypeGeocoderIP = "GEOCODER-IP"

// ConnectionTypeGoeSearch represents a connection that searches addresses / place names
const ConnectionTypeGeoSearch = "GEOSEARCH"

// ConnectionTypeImage represents a connection that can be used to generate images
const ConnectionTypeImage = "IMAGE"

// ConnectionTypeUserUserPayment represents a connection that can take payments for users
const ConnectionTypeUserPayment = "USER-PAYMENT"

/******************************************
 * Provider Definitions
 * definitions of the services that have
 * been implemented in the system.
 ******************************************/

// ConnectionProviderArcGIS represents an API connection to the https://www.arcgis.com service
const ConnectionProviderArcGIS = "ARCGIS"

// ConnectionProviderBing represents an API to the Bing Geocoding service
const ConnectionProviderBing = "BING"

// ConnectionProviderFREEIPAPICOM represents an API connection to the https://freeipapi.com service
const ConnectionProviderFREEIPAPICOM = "FREEIPAPI.COM"

// ConnectionProviderGiphy represents an API connection to the https://giphy.com service
// for generating animated GIFs.
const ConnectionProviderGiphy = "GIPHY"

// ConnectionProviderGoogleMaps represents an API connection to the https://maps.google.com service
const ConnectionProviderGoogleMaps = "GOOGLE-MAPS"

// ConnectionProviderIPAPICO represents an API connection to the https://ipapi.co service
const ConnectionProviderIPAPICO = "IPAPI.CO"

// ConnectionProviderIPAPICOM represents an API connection to the https://ipapi.com service
const ConnectionProviderIPAPICOM = "IP-API.COM"

// ConnectionProviderOpenStreetMap represents an API connection to the https://openstreetmap.org service
const ConnectionProviderOpenStreetMap = "OPENSTREETMAP"

// ConnectionproviderStaticGeocoder represents a static geocoding value that is always the same
const ConnectionProviderStaticGeocoderIP = "STATIC-GEOCODER-IP"

// ConnectionProviderTomTom represents an API connection to the https://tomtom.com mapping service
const ConnectionProviderTomTom = "TOMTOM"

// ConnectionProviderPayPal represents an API connection to the https://paypal.com service
// for processing payments.
// const ConnectionProviderPayPal = "PAYPAL"

// ConnectionProviderStripe represents an API connection to the https://stripe.com service
// for processing payments, using direct API keys.
const ConnectionProviderStripe = "STRIPE"

// ConnectionProviderStripeConnect represents an API connection to the https://stripe.com service
// for processing payments, using the Stripe Connect / OAuth authentication.
const ConnectionProviderStripeConnect = "STRIPE-CONNECT"

// ConnectionProviderUnsplash represents an API connection to the https://unsplash.com service
// for generating photographs.
const ConnectionProviderUnsplash = "UNSPLASH"
