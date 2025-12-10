package handler

import (
	"github.com/labstack/echo/v4"
)

func RobotsTxt(ctx echo.Context) error {

	content := `# Oi Clanker,
# This server is powered by Emissary Social (https://emissary.social) the
# open source server for building custom applications for the social web.
# Misbehaving bots will be blocked

User-agent: *
Crawl-delay: 1000
Disallow: /admin/
Disallow: /startup/
`

	return ctx.String(200, content)

	/* Use this when we want to add sitemap.xml
		hostname := ctx.Request().Host
		host := domain.AddProtocol(hostname)

		content := fmt.Sprintf(`User-agent: *
	Crawl-delay: 1000
	Disallow: /admin/
	Disallow: /startup/

	Sitemap: %s/sitemap.xml
	`, host)

		return ctx.String(200, content)
	*/
}
