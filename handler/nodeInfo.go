package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/labstack/echo/v4"
)

// GetNodeInfo returns the discovery links for nodeInfo endpoints
// http://nodeinfo.diaspora.software/protocol.html
// http://nodeinfo.diaspora.software/schema.html
func GetNodeInfo(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		host := ctx.Request().Host
		server := domain.AddProtocol(host)

		result := map[string]any{
			"links": []map[string]any{
				{
					"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.0",
					"href": server + "/nodeinfo/2.0",
				},
				{
					"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.1",
					"href": server + "/nodeinfo/2.1",
				},
			},
		}

		return ctx.JSON(http.StatusOK, result)
	}
}

// GetNodeInfo20 returns the nodeInfo 2.0 document for this server
// http://nodeinfo.diaspora.software/ns/schema/2.0
// http://nodeinfo.diaspora.software/docson/index.html#/ns/schema/2.0#$$expand
func GetNodeInfo20(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.GetNodeInfo20", "Error loading server factory")
		}

		// Get the Domain
		domainService := factory.Domain()
		domain, err := domainService.LoadDomain()

		if err != nil {
			return derp.Wrap(err, "handler.GetNodeInfo20", "Error loading domain")
		}

		userService := factory.User()
		userCount, _ := userService.Count(exp.All())

		result := map[string]any{
			"version": "2.0",
			"software": map[string]any{
				"name":    "Emissary",
				"version": serverFactory.Version(),
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
}

// GetNodeInfo21 returns the nodeInfo 2.1 document for this server
// http://nodeinfo.diaspora.software/ns/schema/2.1
// http://nodeinfo.diaspora.software/docson/index.html#/ns/schema/2.1#$$expand
func GetNodeInfo21(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.GetNodeInfo20", "Error loading server factory")
		}

		// Get the Domain
		domainService := factory.Domain()
		domain, err := domainService.LoadDomain()

		if err != nil {
			return derp.Wrap(err, "handler.GetNodeInfo20", "Error loading domain")
		}

		userService := factory.User()
		userCount, _ := userService.Count(exp.All())

		result := map[string]any{
			"version": "2.1",
			"software": map[string]any{
				"name":       "Emissary",
				"version":    serverFactory.Version(),
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
}
