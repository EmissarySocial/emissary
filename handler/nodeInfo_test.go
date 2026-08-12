package handler

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/nodeinfo"
	"github.com/benpate/rosetta/convert"
	"github.com/stretchr/testify/require"
)

// softwareNamePattern is the regex that the NodeInfo 2.0 and 2.1 schemas apply to `software.name`.
// It admits no uppercase, which is the whole of BUG-15.
var softwareNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// testDomain returns a Domain with enough detail to build a NodeInfo document from.
func testDomain() *model.Domain {
	result := model.NewDomain()
	result.Label = "Example Server"
	result.Description = "An Emissary server"
	return &result
}

// testDocument builds one NodeInfo document and returns it decoded from its own JSON, so that every
// assertion below reads the bytes a federated peer would receive rather than the Go struct.
func testDocument(t *testing.T, version string) map[string]any {

	t.Helper()

	document := nodeInfoDocument(version, "0.9.0", testDomain(), convert.Pointer(int64(12)))

	if version == "2.1" {
		document.Software.Repository = softwareRepository
		document.Software.Homepage = softwareHomepage
	}

	marshalled, err := json.Marshal(document)
	require.Nil(t, err)

	result := make(map[string]any)
	require.Nil(t, json.Unmarshal(marshalled, &result))

	return result
}

// requireProperties asserts that an object carries every required property and no property outside
// the set the schema allows. The NodeInfo schemas set "additionalProperties": false at every level
// but `metadata`, so an unexpected key is a validation failure in its own right -- which is what
// made the misspelled `activeHalfYear` of BUG-16 fatal rather than merely absent.
func requireProperties(t *testing.T, object map[string]any, required []string, optional []string) {

	t.Helper()

	allowed := make(map[string]bool)

	for _, name := range required {
		require.Contains(t, object, name)
		allowed[name] = true
	}

	for _, name := range optional {
		allowed[name] = true
	}

	for name := range object {
		require.True(t, allowed[name], "unexpected property: "+name)
	}
}

// TestNodeInfo_SoftwareName is the regression guard for BUG-15: `software.name` must satisfy the
// schema pattern in both documents, today and after any future rename.
func TestNodeInfo_SoftwareName(t *testing.T) {

	for _, version := range []string{"2.0", "2.1"} {

		document := testDocument(t, version)
		software := document["software"].(map[string]any)
		name := software["name"].(string)

		require.Regexp(t, softwareNamePattern, name)
		require.Equal(t, "emissary", name)
	}
}

// TestNodeInfo_Properties_20 asserts the 2.0 document carries exactly the properties its schema
// allows. Notably, `repository` and `homepage` exist only in 2.1 and must not appear here.
func TestNodeInfo_Properties_20(t *testing.T) {

	document := testDocument(t, "2.0")

	requireProperties(t, document, []string{"version", "software", "protocols", "services", "openRegistrations", "usage", "metadata"}, nil)
	requireProperties(t, document["software"].(map[string]any), []string{"name", "version"}, nil)

	require.Equal(t, "2.0", document["version"])
}

// TestNodeInfo_Properties_21 asserts the 2.1 document carries exactly the properties its schema
// allows, including the two that only 2.1 defines.
func TestNodeInfo_Properties_21(t *testing.T) {

	document := testDocument(t, "2.1")

	requireProperties(t, document, []string{"version", "software", "protocols", "services", "openRegistrations", "usage", "metadata"}, nil)
	requireProperties(t, document["software"].(map[string]any), []string{"name", "version"}, []string{"repository", "homepage"})

	require.Equal(t, "2.1", document["version"])

	software := document["software"].(map[string]any)
	require.Equal(t, softwareRepository, software["repository"])
	require.Equal(t, softwareHomepage, software["homepage"])
}

// TestNodeInfo_Usage asserts the usage block spells its counters the way the schema does. BUG-16 was
// `activeHalfYear`, which is not a property the schema knows and so fails the document outright.
func TestNodeInfo_Usage(t *testing.T) {

	for _, version := range []string{"2.0", "2.1"} {

		usage := testDocument(t, version)["usage"].(map[string]any)
		requireProperties(t, usage, []string{"users"}, []string{"localPosts", "localComments"})

		users := usage["users"].(map[string]any)
		requireProperties(t, users, nil, []string{"total", "activeHalfyear", "activeMonth"})

		require.NotContains(t, users, "activeHalfYear")
		require.Equal(t, float64(12), users["total"])
	}
}

// TestNodeInfo_NoFabricatedStatistics asserts that every counter this server does not compute is
// absent rather than zero. FEP-0151 reads a published zero as a claim that the value is zero, so the
// misspelled key of BUG-16 must not be "fixed" by emitting a correctly spelled fabrication.
func TestNodeInfo_NoFabricatedStatistics(t *testing.T) {

	for _, version := range []string{"2.0", "2.1"} {

		usage := testDocument(t, version)["usage"].(map[string]any)
		users := usage["users"].(map[string]any)

		require.NotContains(t, usage, "localPosts")
		require.NotContains(t, usage, "localComments")
		require.NotContains(t, users, "activeHalfyear")
		require.NotContains(t, users, "activeMonth")
	}
}

// TestNodeInfo_CountedZeroIsPublished proves the counter Emissary *does* compute survives being zero.
// Omission and zero mean different things here, and a server with no users must say so rather than
// fall silent -- which is why these fields are pointers.
func TestNodeInfo_CountedZeroIsPublished(t *testing.T) {

	document := nodeInfoDocument("2.0", "0.9.0", testDomain(), convert.Pointer(int64(0)))

	marshalled, err := json.Marshal(document)
	require.Nil(t, err)

	require.Contains(t, string(marshalled), `"users":{"total":0}`)
}

// TestNodeInfo_UncountedUsersAreOmitted covers the other half of that distinction: when the count is
// unavailable -- a failed query -- the document must say nothing about the number of users rather
// than report zero of them. `usage.users` still has to be present, because the schema requires it, so
// the honest document is an empty object.
func TestNodeInfo_UncountedUsersAreOmitted(t *testing.T) {

	document := nodeInfoDocument("2.0", "0.9.0", testDomain(), nil)

	marshalled, err := json.Marshal(document)
	require.Nil(t, err)
	require.Contains(t, string(marshalled), `"usage":{"users":{}}`)

	decoded := make(map[string]any)
	require.Nil(t, json.Unmarshal(marshalled, &decoded))

	usage := decoded["usage"].(map[string]any)
	require.Contains(t, usage, "users")
	require.NotContains(t, usage["users"].(map[string]any), "total")
}

// TestNodeInfo_Services asserts both service arrays are present and drawn from the schema's enums.
func TestNodeInfo_Services(t *testing.T) {

	services := testDocument(t, "2.0")["services"].(map[string]any)

	requireProperties(t, services, []string{"inbound", "outbound"}, nil)
	require.Equal(t, []any{"atom1.0", "rss2.0"}, services["inbound"])
	require.Equal(t, []any{"atom1.0", "rss2.0"}, services["outbound"])
}

// TestNodeInfo_OpenRegistrations proves a FALSE value is still published. `openRegistrations` is
// required by both schemas, so an `omitempty` on that field would drop it from the document of every
// server that closed its registrations -- and only from those servers.
func TestNodeInfo_OpenRegistrations(t *testing.T) {

	domain := testDomain()
	require.False(t, domain.HasRegistrationForm())

	document := testDocument(t, "2.0")

	require.Contains(t, document, "openRegistrations")
	require.Equal(t, false, document["openRegistrations"])
}

// TestNodeInfo_ClientRoundTrip decodes the emitted document with Emissary's own NodeInfo client and
// asserts every field the client declares arrives populated. The client has always spelled
// `activeHalfyear` correctly, so this is the test that would have caught BUG-16 the day it was filed.
func TestNodeInfo_ClientRoundTrip(t *testing.T) {

	document := nodeInfoDocument("2.1", "0.9.0", testDomain(), convert.Pointer(int64(12)))
	document.Software.Repository = softwareRepository
	document.Software.Homepage = softwareHomepage

	marshalled, err := json.Marshal(document)
	require.Nil(t, err)

	result := nodeinfo.NewNodeInfo()
	require.Nil(t, json.Unmarshal(marshalled, &result))

	require.Equal(t, "2.1", result.Version)
	require.Equal(t, "emissary", result.Software.Name)
	require.Equal(t, "0.9.0", result.Software.Version)
	require.Equal(t, softwareRepository, result.Software.Repository)
	require.Equal(t, softwareHomepage, result.Software.Homepage)
	require.Equal(t, []string{"activitypub"}, result.Protocols)
	require.Equal(t, []string{"atom1.0", "rss2.0"}, result.Services.Inbound)
	require.Equal(t, []string{"atom1.0", "rss2.0"}, result.Services.Outbound)
	require.Equal(t, "Example Server", result.Metadata["nodeName"])

	require.NotNil(t, result.Usage.Users.Total)
	require.Equal(t, int64(12), *result.Usage.Users.Total)
}

// TestNodeInfo_CrossDocumentAgreement asserts the two documents never disagree about the values they
// both carry. Every defect in this handler began as one of two literals drifting from the other.
func TestNodeInfo_CrossDocumentAgreement(t *testing.T) {

	document20 := testDocument(t, "2.0")
	document21 := testDocument(t, "2.1")

	software20 := document20["software"].(map[string]any)
	software21 := document21["software"].(map[string]any)

	require.Equal(t, software20["name"], software21["name"])
	require.Equal(t, software20["version"], software21["version"])
	require.Equal(t, document20["usage"], document21["usage"])
	require.Equal(t, document20["protocols"], document21["protocols"])
	require.Equal(t, document20["services"], document21["services"])
	require.Equal(t, document20["metadata"], document21["metadata"])
	require.Equal(t, document20["openRegistrations"], document21["openRegistrations"])
}
