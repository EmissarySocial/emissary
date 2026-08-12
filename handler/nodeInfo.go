package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/nodeinfo"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// softwareName is the canonical NodeInfo identifier for this server. The NodeInfo 2.0/2.1 schemas
// constrain software.name to ^[a-z0-9-]+$ -- no uppercase -- and the value is a join key across the
// ecosystem (the-federation.info, fediverse.observer, and Emissary's own tools/camper). It is NOT a
// display name; metadata.nodeName carries domain.Label for that.
const softwareName = "emissary"

// softwareRepository is the source code repository that NodeInfo 2.1 publishes. NodeInfo 2.0 has no
// such property and rejects it as an unexpected one.
const softwareRepository = "https://github.com/EmissarySocial/emissary"

// softwareHomepage is the project homepage that NodeInfo 2.1 publishes, under the same 2.0 caveat.
const softwareHomepage = "https://emissary.social"

// GetNodeInfo returns the discovery links for nodeInfo endpoints
// http://nodeinfo.diaspora.software/protocol.html
// http://nodeinfo.diaspora.software/schema.html
func GetNodeInfo(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	hostname := factory.Hostname()
	server := uri.PrependProtocol(hostname)

	result := map[string]any{
		"links": []map[string]any{
			{
				"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.0",
				"href": server + "/.well-known/nodeinfo/2.0",
			},
			{
				"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.1",
				"href": server + "/.well-known/nodeinfo/2.1",
			},
			{
				"rel":  "https://www.w3.org/ns/activitystreams#Application",
				"href": server + "/@application",
			},
		},
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetNodeInfo20 returns the nodeInfo 2.0 document for this server
// http://nodeinfo.diaspora.software/ns/schema/2.0
// http://nodeinfo.diaspora.software/docson/index.html#/ns/schema/2.0#$$expand
func GetNodeInfo20(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	domain := factory.Domain().Get()
	userCount := nodeInfoUserCount(factory, session)

	return ctx.JSON(http.StatusOK, nodeInfoDocument("2.0", factory.Version(), domain, userCount))
}

// GetNodeInfo21 returns the nodeInfo 2.1 document for this server
// http://nodeinfo.diaspora.software/ns/schema/2.1
// http://nodeinfo.diaspora.software/docson/index.html#/ns/schema/2.1#$$expand
func GetNodeInfo21(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	domain := factory.Domain().Get()
	userCount := nodeInfoUserCount(factory, session)

	result := nodeInfoDocument("2.1", factory.Version(), domain, userCount)

	// Only 2.1 defines these two properties
	result.Software.Repository = softwareRepository
	result.Software.Homepage = softwareHomepage

	return ctx.JSON(http.StatusOK, result)
}

// nodeInfoUserCount counts the Users registered on this server, or returns NIL if they cannot be
// counted. NIL publishes no user count at all, which is the only honest answer to a failed query.
func nodeInfoUserCount(factory *service.Factory, session data.Session) *int64 {

	const location = "handler.nodeInfoUserCount"

	userCount, err := factory.User().Count(session, exp.All())

	// RULE: A server that cannot count its Users must not claim to have none (FEP-0151)
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Counting Users for NodeInfo"))
		return nil
	}

	return convert.Pointer(userCount)
}

// nodeInfoDocument builds the NodeInfo properties that the 2.0 and 2.1 documents share. Callers add
// the properties that only their own version defines.
func nodeInfoDocument(version string, softwareVersion string, domain *model.Domain, userCount *int64) nodeinfo.NodeInfo {

	result := nodeinfo.NewNodeInfo()
	result.Version = version

	// Identify the software. Both values are machine-readable identifiers, not display text.
	result.Software = nodeinfo.SoftwareInfo{
		Name:    softwareName,
		Version: softwareVersion,
	}

	// Describe what this server speaks
	result.Protocols = []string{"activitypub"}
	result.Services = nodeinfo.ServicesInfo{
		Inbound:  []string{"atom1.0", "rss2.0"},
		Outbound: []string{"atom1.0", "rss2.0"},
	}
	result.OpenRegistrations = domain.HasRegistrationForm()

	// RULE: FEP-0151 forbids publishing skewed statistics, so a counter this server does not actually
	// compute is omitted rather than published as a fabricated zero. Omission says "not disclosed";
	// a zero says "none exist". `total` is the only statistic Emissary counts, and it too drops out
	// when the count is unavailable.
	result.Usage = nodeinfo.UsageInfo{
		Users: nodeinfo.UsersInfo{
			Total: userCount,
		},
	}

	// Describe this Domain to humans
	result.Metadata["nodeName"] = domain.Label
	result.Metadata["nodeDescription"] = domain.Description

	return result
}
