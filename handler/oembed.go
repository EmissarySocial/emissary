package handler

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

// GetOEmbed will provide an OEmbed service to be used exclusively by websites on this domain.
func GetOEmbed(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetOEmbed"

	// Verify that the URL is valid
	token := ctx.QueryParam("url")
	format := ctx.QueryParam("format")

	parsedToken, err := url.Parse(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid URL")
	}

	// Verify that the URL is on this domain
	if parsedToken.Host != factory.Hostname() {
		return derp.NotFoundError(location, "Invalid URL", "URL does not match domain")
	}

	// Load the OEmbed result
	result, err := getOEmbed_record(factory, session, parsedToken.Path)

	if err != nil {
		return derp.Wrap(err, location, "Error loading OEmbed record")
	}

	// Return the result in the requested format
	if format == "xml" {
		return ctx.XML(200, result)
	}

	return ctx.JSON(200, result)
}

func getOEmbed_record(factory *service.Factory, session data.Session, path string) (mapof.Any, error) {

	// Parse the path as either a Stream or a User
	path = strings.TrimPrefix(path, "/")

	// If the path is empty, then return oEmbed for the Domain
	if path == "" {
		return getOEmbed_Domain(factory)
	}

	// If the path begins with "@", then it is a User
	if strings.HasPrefix(path, "@") {
		path = strings.TrimPrefix(path, "@")
		return getOEmbed_User(factory, session, path)
	}

	// Otherwise, the path is for a Stream
	return getOEmbed_Stream(factory, session, path)
}

func getOEmbed_Domain(factory *service.Factory) (mapof.Any, error) {

	domain := factory.Domain().Get()

	result := mapof.Any{
		"version":       "1.0",
		"type":          "link",
		"title":         domain.Label,
		"cache_age":     86400, // cache for 24 hours
		"provider_name": domain.Label,
		"provider_url":  domain.Host(),
	}

	return result, nil
}

func getOEmbed_Stream(factory *service.Factory, session data.Session, token string) (mapof.Any, error) {

	const location = "handler.getOEmbed_Stream"

	// Load the Stream
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByToken(session, token, &stream); err != nil {
		return mapof.Any{}, derp.Wrap(err, location, "Error loading stream from database")
	}

	// Export the stream as an OEmbed object
	// Export the user as an OEmbed object
	// Get the domain
	domain := factory.Domain().Get()

	// Export the user as an OEmbed object
	result := mapof.Any{
		"version":       "1.0",
		"type":          "link",
		"title":         stream.Label,
		"cache_age":     86400, // cache for 24 hours
		"provider_name": domain.Label,
		"provider_url":  domain.Host(),
	}

	if iconURL := stream.IconURL; iconURL != "" {
		result["thumbnail_url"] = iconURL + ".webp?height=300&width=300"
		result["thumbnail_height"] = 300
		result["thumbnail_width"] = 300
	}

	/* This works great, but I'm removing it for not because Mastodon doesn't
	   support "rich" style oEmbed.

	// Special case for Templates that define HTML content of OEmbed
	templateService := factory.Template()
	if template, err := templateService.Load(stream.TemplateID); err == nil {

		if htmlTemplate := template.GetOEmbed(); htmlTemplate != nil {

			if builder, err := build.NewStream(factory, ctx.Request(), ctx.Response(), template, &stream, "view"); err == nil {

				html := executeTemplate(htmlTemplate, builder)

				if html != "" {

					// Enable this line for nice-ish debugging
					// return nil, ctx.HTML(200, html)

					result["html"] = html
					result["type"] = "rich"

					height, width := getOEmbed_heightAndWidth(html)

					if height > 0 {
						result["height"] = height
					}

					if width > 0 {
						result["width"] = width
					}
				}
			}
		}
	}
	*/

	return result, nil
}

func getOEmbed_User(factory *service.Factory, session data.Session, token string) (mapof.Any, error) {

	const location = "handler.getOEmbed_User"

	// Load the User
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByToken(session, token, &user); err != nil {
		return mapof.Any{}, derp.Wrap(err, location, "Error loading user from database")
	}

	// Get the domain
	domain := factory.Domain().Get()

	// Export the user as an OEmbed object
	result := mapof.Any{
		"version":       "1.0",
		"type":          "link",
		"title":         "@" + domain.Hostname + "@" + user.Username,
		"cache_age":     86400, // cache for 24 hours
		"provider_name": domain.Label,
		"provider_url":  domain.Host(),
	}

	if iconURL := user.ActivityPubIconURL(); iconURL != "" {
		result["thumbnail_url"] = iconURL + ".webp?height=300&width=300"
		result["thumbnail_height"] = 300
		result["thumbnail_width"] = 300
	}

	return result, nil
}

/*
//nolint:unused
func getOEmbed_heightAndWidth(html string) (int, int) {

	var height int
	var width int

	// Find height
	findHeight := regexp.MustCompile(`height:\s*(\d+)px;`)
	heightStrings := findHeight.FindStringSubmatch(html)

	if len(heightStrings) == 2 {
		height = convert.Int(heightStrings[1])
	}

	// Find width
	findWidth := regexp.MustCompile(`width:\s*(\d+)px;`)
	widthStrings := findWidth.FindStringSubmatch(html)

	if len(widthStrings) == 2 {
		width = convert.Int(widthStrings[1])
	}

	return height, width
}
*/
