package geocoder

import (
	"testing"

	"github.com/benpate/geo"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Constructor and capability tests
 *
 * Each provider supports a different subset of the four geocoding
 * interfaces. The assertions below are compile-time, so a method
 * removed or renamed breaks the build instead of a caller.
 ******************************************/

// These pin which provider offers which capability. A provider that loses a method
// fails to compile here rather than silently failing an interface check at runtime.
var (
	_ AddressGeocoder = Here{}
	_ AddressGeocoder = Nominatim{}
	_ AddressGeocoder = GoogleMaps{}
	_ AddressGeocoder = Geoapify{}
	_ AddressGeocoder = Maptiler{}
	_ AddressGeocoder = Geocodio{}
	_ AddressGeocoder = OpenStreetMap{}
	_ AddressGeocoder = Nil{}

	_ NetworkGeocoder = Geoapify{}
	_ NetworkGeocoder = IPAPICOM{}
	_ NetworkGeocoder = FREEIPAPICOM{}
	_ NetworkGeocoder = Static{}
	_ NetworkGeocoder = Nil{}

	_ TimezoneGeocoder = Here{}
	_ TimezoneGeocoder = GoogleMaps{}
	_ TimezoneGeocoder = Geoapify{}
	_ TimezoneGeocoder = Geocodio{}
	_ TimezoneGeocoder = Nil{}

	_ AddressAutocompleter = Here{}
	_ AddressAutocompleter = Nominatim{}
	_ AddressAutocompleter = GoogleMaps{}
	_ AddressAutocompleter = Geoapify{}
	_ AddressAutocompleter = Maptiler{}
	_ AddressAutocompleter = Nil{}
)

// TestNewNominatim_DefaultURL pins the only branch in any constructor: an empty
// searchURL falls back to the public OpenStreetMap Nominatim server.
func TestNewNominatim_DefaultURL(t *testing.T) {

	geocoder := NewNominatim("", "key", "agent", "referer")

	require.Equal(t, "https://nominatim.openstreetmap.org", geocoder.searchURL)
	require.Equal(t, "key", geocoder.apiKey)
	require.Equal(t, "agent", geocoder.userAgent)
	require.Equal(t, "referer", geocoder.referer)
}

// TestNewNominatim_CustomURL confirms a self-hosted Nominatim URL is kept verbatim,
// with no trailing-slash or scheme rewriting.
func TestNewNominatim_CustomURL(t *testing.T) {

	for _, searchURL := range []string{
		"https://nominatim.example.com",
		"https://nominatim.example.com/",
		"http://nominatim.example.com:8080",
	} {
		require.Equal(t, searchURL, NewNominatim(searchURL, "", "", "").searchURL)
	}
}

// TestConstructors_StoreCredentials confirms every provider keeps the credentials it
// was given, which is the whole job of these constructors.
func TestConstructors_StoreCredentials(t *testing.T) {

	require.Equal(t, "geoapify-key", NewGeoapify("geoapify-key").apiKey)
	require.Equal(t, "geocodio-key", NewGeocodio("geocodio-key").apiKey)
	require.Equal(t, "google-key", NewGoogleMaps("google-key").apiKey)
	require.Equal(t, "maptiler-key", NewMaptiler("maptiler-key").apiKey)
	require.Equal(t, "ipapi-key", NewIPAPICOM("ipapi-key").apiKey)
	require.Equal(t, "freeipapi-key", NewFREEIPAPICOM("freeipapi-key").apiKey)

	here := NewHere("here-id", "here-key")
	require.Equal(t, "here-id", here.apiID)
	require.Equal(t, "here-key", here.apiKey)
}

// TestConstructors_EmptyCredentials confirms an unconfigured provider still constructs.
// Validation happens at the API call, not here, so a missing key is not a panic.
func TestConstructors_EmptyCredentials(t *testing.T) {

	require.Equal(t, "", NewGeoapify("").apiKey)
	require.Equal(t, "", NewGeocodio("").apiKey)
	require.Equal(t, "", NewGoogleMaps("").apiKey)
	require.Equal(t, "", NewMaptiler("").apiKey)
	require.Equal(t, "", NewIPAPICOM("").apiKey)
	require.Equal(t, "", NewFREEIPAPICOM("").apiKey)
	require.Equal(t, "", NewHere("", "").apiKey)
}

// TestNewStatic_StoresCoordinates confirms Static keeps latitude and longitude in the
// order the constructor names them, which is the reverse of geo.NewPoint's order.
func TestNewStatic_StoresCoordinates(t *testing.T) {

	geocoder := NewStatic(39.7392, -104.9903)

	require.Equal(t, 39.7392, geocoder.latitude)
	require.Equal(t, -104.9903, geocoder.longitude)
}

// TestNewZeroArgConstructors confirms the stateless providers construct to their zero value.
func TestNewZeroArgConstructors(t *testing.T) {

	require.Equal(t, Nil{}, NewNil())
	require.Equal(t, OpenStreetMap{}, NewOpenStreetMap())
}

/******************************************
 * Offline error paths
 *
 * benpate/remote refuses to dial a non-public address, and no geocoder
 * opts out. Pointing a request at loopback therefore fails in the
 * dialer, which exercises the error branches with no network at all.
 ******************************************/

// blockedURL is a loopback address that remote's private-IP guard rejects before
// any connection is attempted, so these tests are deterministic and offline.
const blockedURL = "http://127.0.0.1:1"

// TestNominatim_GeocodeAddress_TransportError confirms a failed request is wrapped and
// returned, rather than surfacing as an empty Address with a nil error.
func TestNominatim_GeocodeAddress_TransportError(t *testing.T) {

	address, err := NewNominatim(blockedURL, "", "test-agent", "").GeocodeAddress("Denver")

	require.NotNil(t, err)
	require.True(t, address.IsZero())
}

// TestNominatim_AutocompleteAddress_TransportError confirms the autocomplete path also
// reports the failure, and returns no partial suggestions.
func TestNominatim_AutocompleteAddress_TransportError(t *testing.T) {

	results, err := NewNominatim(blockedURL, "", "test-agent", "").AutocompleteAddress("Denv", geo.Point{})

	require.NotNil(t, err)
	require.Equal(t, 0, len(results))
}

// TestNominatim_AutocompleteAddress_WithBias confirms a non-zero bias point does not
// change the failure behavior, since Nominatim ignores bias entirely.
func TestNominatim_AutocompleteAddress_WithBias(t *testing.T) {

	_, err := NewNominatim(blockedURL, "", "test-agent", "").AutocompleteAddress("Denv", geo.NewPoint(-104.9903, 39.7392))

	require.NotNil(t, err)
}
