package build

import (
	"bytes"
	"context"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// backgroundRequestKey marks a request that draws the page behind a modal (as-modal's background-url)
type backgroundRequestKey struct{}

// isBackgroundRequest returns TRUE if this request draws the page behind another modal
func isBackgroundRequest(request *http.Request) bool {
	marked, _ := request.Context().Value(backgroundRequestKey{}).(bool)
	return marked
}

// newBackgroundBuilder returns the builder for a local URL, as the router would build it for a full-page
// GET: "/@{user}" and "/@{user}/{action}" are a User's Outbox, and "/{stream}" and "/{stream}/{action}" are
// a Stream.  The visitor's own session and cookies are kept, so the background checks permissions exactly
// as if the visitor had opened it, and its query string is what the background's QueryParam reads.
func newBackgroundBuilder(builder Builder, target string) (Builder, error) {

	const location = "build.newBackgroundBuilder"

	// RULE: Only a path on this server can be a background
	if !isLocalPath(target) {
		return nil, derp.BadRequest(location, "Background must be a local path", target)
	}

	parsed, err := url.Parse(target)

	if err != nil {
		return nil, derp.BadRequest(location, "Invalid background URL", target, err.Error())
	}

	token, actionID, ok := parseBackgroundPath(parsed.Path)

	if !ok {
		return nil, derp.BadRequest(location, "Background must name one User or Stream, and an optional action", target)
	}

	// The background's request is the visitor's, at the new URL, marked so that it cannot draw a modal of its own
	original := builder.request()
	request := original.Clone(context.WithValue(original.Context(), backgroundRequestKey{}, true))
	request.URL.Path = parsed.Path
	request.URL.RawPath = parsed.RawPath
	request.URL.RawQuery = parsed.RawQuery
	request.RequestURI = parsed.RequestURI()

	factory := builder.factory()
	session := builder.session()

	// "/@{user}" is a User's Outbox
	if username, isUser := strings.CutPrefix(token, "@"); isUser {

		user := model.NewUser()

		if err := factory.User().LoadByToken(session, username, &user); err != nil {
			return nil, derp.Wrap(err, location, "Loading User", username)
		}

		outbox, err := NewOutbox(factory, session, request, builder.response(), &user, actionID)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating Outbox builder", username)
		}

		return outbox, nil
	}

	// Anything else is a Stream
	stream := model.NewStream()

	if err := factory.Stream().LoadByToken(session, token, &stream); err != nil {
		return nil, derp.Wrap(err, location, "Loading Stream", token)
	}

	streamBuilder, err := NewStreamWithoutTemplate(factory, session, request, builder.response(), &stream, actionID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Creating Stream builder", token)
	}

	return streamBuilder, nil
}

// parseBackgroundPath splits "/{token}" or "/{token}/{action}" into its token and action, which defaults
// to "view".  ok is FALSE for any other shape.
func parseBackgroundPath(path string) (token string, actionID string, ok bool) {

	switch segments := strings.Split(strings.TrimPrefix(path, "/"), "/"); {
	case len(segments) == 1 && segments[0] != "":
		return segments[0], "view", true

	case len(segments) == 2 && segments[0] != "" && segments[1] != "":
		return segments[0], segments[1], true
	}

	return "", "", false
}

// renderBackground runs a builder's action and wraps it in the theme's page, as AsHTML does for a full page
func renderBackground(backgroundBuilder Builder) (string, error) {

	const location = "build.renderBackground"

	var partialPage bytes.Buffer
	factory := backgroundBuilder.factory()
	pipeline := Pipeline(backgroundBuilder.action().Steps)

	status := pipeline.Execute(factory, backgroundBuilder, &partialPage, ActionMethodGet)

	if status.Error != nil {
		return "", derp.Wrap(status.Error, location, "Building background")
	}

	// RULE: A background that redirects (a collection's view forwards to its profile tab) has no page to
	// draw, and its Location header must never reach the response
	if status.Headers["Location"] != "" {
		return "", derp.BadRequest(location, "Background redirects elsewhere", status.Headers["Location"])
	}

	// Copy status values into the Response...
	status.Apply(backgroundBuilder.response())

	// Full Page requests require the theme service to wrap the built content
	var fullPage bytes.Buffer
	htmlTemplate := factory.Domain().Theme().HTMLTemplate
	backgroundBuilder.SetContent(partialPage.String())

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", backgroundBuilder); err != nil {
		return "", derp.Wrap(err, location, "Executing template")
	}

	return fullPage.String(), nil
}

// insertHead puts the modal's <head> values into a full page, just before </head>.  When they carry a
// <title>, the page's own <title> is removed first, so the page has only one.
func insertHead(page string, head string) string {

	if strings.TrimSpace(head) == "" {
		return page
	}

	headEnd := strings.Index(page, "</head>")

	if headEnd < 0 {
		return page
	}

	if strings.Contains(head, "<title") {
		page, headEnd = removeTitle(page, headEnd)
	}

	return page[:headEnd] + head + page[headEnd:]
}

// removeTitle removes the first <title> element that ends before headEnd, and returns the page with
// headEnd moved to match
func removeTitle(page string, headEnd int) (string, int) {

	titleStart := strings.Index(page[:headEnd], "<title")

	if titleStart < 0 {
		return page, headEnd
	}

	titleLength := strings.Index(page[titleStart:headEnd], "</title>")

	if titleLength < 0 {
		return page, headEnd
	}

	titleEnd := titleStart + titleLength + len("</title>")

	return page[:titleStart] + page[titleEnd:], headEnd - (titleEnd - titleStart)
}

// markReturnURL gives a modal drawn on a full page the address of the page behind it, so that the
// theme's modal-url.js can show that address when the modal closes
func markReturnURL(modal string, returnURL string) string {

	if returnURL == "" {
		return modal
	}

	const modalBegin = `<div id="modal"`
	return strings.Replace(modal, modalBegin, modalBegin+` data-return-url="`+html.EscapeString(returnURL)+`"`, 1)
}

// isLocalPath returns TRUE if a URL is a path on this server: it begins with one slash, and names no
// scheme or host.  "//host" and "/\host" are read by browsers as another host, so neither is local.
func isLocalPath(target string) bool {

	if !strings.HasPrefix(target, "/") {
		return false
	}

	if strings.HasPrefix(target, "//") || strings.HasPrefix(target, "/\\") {
		return false
	}

	return true
}
