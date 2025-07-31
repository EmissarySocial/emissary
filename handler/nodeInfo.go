package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/data"
	domaintools "github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/steranko"
)

// GetNodeInfo returns the discovery links for nodeInfo endpoints
// http://nodeinfo.diaspora.software/protocol.html
// http://nodeinfo.diaspora.software/schema.html
func GetNodeInfo(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	host := domaintools.Hostname(ctx.Request())
	server := domaintools.AddProtocol(host)

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
func GetNodeInfo20(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	// Get the Domain
	domain := factory.Domain().Get()

	userService := factory.User()
	userCount, _ := userService.Count(session, exp.All())

	result := map[string]any{
		"version": "2.0",
		"software": map[string]any{
			"name":    "Emissary",
			"version": factory.Version(),
		},
		"protocols": []string{"activitypub"},
		"services": map[string]any{
			"inbound":  []string{"atom1.0", "rss2.0"},
			"outbound": []string{"atom1.0", "rss2.0"},
		},
		"openRegistrations": domain.HasRegistrationForm(),
		"usage": map[string]any{
			"users": map[string]any{
				"total":          userCount,
				"activeHalfYear": 0,
				"activeMonth":    0,
			},
			"localPosts":    0,
			"localComments": 0,
		},
		"metadata": map[string]any{
			"nodeName":        domain.Label,
			"nodeDescription": domain.Description,
		},
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetNodeInfo21 returns the nodeInfo 2.1 document for this server
// http://nodeinfo.diaspora.software/ns/schema/2.1
// http://nodeinfo.diaspora.software/docson/index.html#/ns/schema/2.1#$$expand
func GetNodeInfo21(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	// Get the Domain
	domain := factory.Domain().Get()

	userService := factory.User()
	userCount, _ := userService.Count(session, exp.All())

	result := map[string]any{
		"version": "2.1",
		"software": map[string]any{
			"name":       "Emissary",
			"version":    factory.Version(),
			"repository": "https://github.com/EmissarySocial/emissary",
			"homepage":   "https://emissary.social",
		},
		"protocols": []string{"activitypub"},
		"services": map[string]any{
			"inbound":  []string{"atom1.0", "rss2.0"},
			"outbound": []string{"atom1.0", "rss2.0"},
		},
		"openRegistrations": domain.HasRegistrationForm(),
		"usage": map[string]any{
			"users": map[string]any{
				"total":          userCount,
				"activeHalfYear": 0,
				"activeMonth":    0,
			},
			"localPosts":    0,
			"localComments": 0,
		},
		"metadata": map[string]any{
			"nodeName":        domain.Label,
			"nodeDescription": domain.Description,
		},
	}

	return ctx.JSON(http.StatusOK, result)
}
