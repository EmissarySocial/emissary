package server

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/require"
)

// testPersonalizedFactory returns a factoryCore whose registry serves the named hostnames
func testPersonalizedFactory(hostnames ...string) *factoryCore {

	factory := &factoryCore{}
	factory.domains = xsync.NewMap[string, *service.Factory]()

	for _, hostname := range hostnames {
		factory.domains.Store(hostname, &service.Factory{})
	}

	return factory
}

// TestByPersonalizedHostname_Resolves verifies the hostnames that DO name a parent Domain
func TestByPersonalizedHostname_Resolves(t *testing.T) {

	factory := testPersonalizedFactory("bandwagon.fm")

	table := []struct {
		hostname string
		username string
	}{
		{"artist.bandwagon.fm", "artist"},
		{"ARTIST.BANDWAGON.FM", "artist"},      // normalized to lowercase
		{"artist.bandwagon.fm:8080", "artist"}, // port is stripped
		{"www.artist.bandwagon.fm", "artist"},  // leading "www." is stripped
		{"artist.www.bandwagon.fm", "artist"},  // parent is normalized too
	}

	for _, test := range table {

		parentFactory, username, err := factory.ByPersonalizedHostname(test.hostname)

		require.Nil(t, err, test.hostname)
		require.NotNil(t, parentFactory, test.hostname)
		require.Equal(t, test.username, username, test.hostname)
	}
}

// TestByPersonalizedHostname_Refuses verifies that a hostname with no served parent
// Domain answers 421 instead of being forwarded as a username
func TestByPersonalizedHostname_Refuses(t *testing.T) {

	factory := testPersonalizedFactory("bandwagon.fm")

	table := []string{
		"localhost",                            // no parent hostname at all
		"localhost:8080",                       // ...still none once the port is stripped
		"",                                     // empty Host header
		"bandwagon.fm",                         // the parent would be "fm"
		"plankton-app.ondigitalocean.app",      // parent is not served here
		"artist.bandwagon.fm.evil.example.com", // the served name is not the TAIL
	}

	for _, hostname := range table {

		parentFactory, username, err := factory.ByPersonalizedHostname(hostname)

		require.NotNil(t, err, hostname)
		require.Nil(t, parentFactory, hostname)
		require.Zero(t, username, hostname)
		require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err), hostname)
	}
}

// TestByPersonalizedHostname_CannotDetectAbandonedDomain pins the reason handler.GetHome
// must ALSO confirm that the username exists.  A subdomain that dropped out of the registry
// is spelled exactly like a personalized domain, so this resolver accepts both (BUG-142).
func TestByPersonalizedHostname_CannotDetectAbandonedDomain(t *testing.T) {

	factory := testPersonalizedFactory("emissary.social")

	// "atlasdemo.emissary.social" was its own Domain until it left the configuration
	parentFactory, username, err := factory.ByPersonalizedHostname("atlasdemo.emissary.social")

	require.Nil(t, err)
	require.NotNil(t, parentFactory)
	require.Equal(t, "atlasdemo", username)
}
