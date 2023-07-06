package render

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// WrapInlineSuccess sends a confirmation message to the #inline-confirmation element
func WrapInlineSuccess(ctx echo.Context, message any) error {

	ctx.Response().Header().Set("HX-Reswap", "innerHTML")
	ctx.Response().Header().Set("HX-Retarget", "#htmx-response-message")

	return ctx.HTML(http.StatusOK, `<span class="green">`+convert.String(message)+`</span>`)
}

// WrapInlineError sends a confirmation message to the #inline-confirmation element
func WrapInlineError(ctx echo.Context, err error) error {

	ctx.Response().Header().Set("HX-Reswap", "innerHTML")
	ctx.Response().Header().Set("HX-Retarget", "#htmx-response-message")

	// nolint:errcheck
	if derpError, ok := err.(derp.SingleError); ok {
		derp.Report(derpError)
		return ctx.HTML(http.StatusOK, `<span class="red">`+derpError.Message+`</span>`)
	}

	// nolint:errcheck
	derp.Report(err)
	return ctx.HTML(http.StatusOK, `<span class="red">`+derp.Message(err)+`</span>`)
}

func WrapModal(response *echo.Response, content string, options ...string) string {

	// These three headers make it a modal
	header := response.Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Reswap", "innerHTML")
	header.Set("HX-Push", "false")

	optionMap := parseOptions(options...)

	// Build the HTML
	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal").Script("install Modal").Data("hx-swap", "none")
	b.Div().ID("modal-underlay").Close()
	b.Div().ID("modal-window").Class(optionMap.GetString("class")).EndBracket() // this is needed because we're embedding foreign content below.

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalWithCloseButton(response *echo.Response, content string, options ...string) string {
	b := html.New()

	b.Div()
	b.Button().Script("on click trigger closeModal").InnerText("Close Window")

	return WrapModal(response, content+b.String())
}

func WrapTooltip(response *echo.Response, content string) string {

	// These headers make it a modal
	header := response.Header()
	header.Set("HX-Reswap", "beforeend")
	header.Set("HX-Push", "false")

	b := html.New()

	b.Span().ID("tooltip").Script("install tooltip").EndBracket()
	b.WriteString(content)
	b.CloseAll()

	return b.String()
}

func WrapForm(endpoint string, content string, options ...string) string {

	optionMap := parseOptions(options...)

	// Allow options to override the endpoint
	if optionEndpoint := optionMap.GetString("endpoint"); optionEndpoint != "" {
		endpoint = optionEndpoint
	}

	b := html.New()

	// Form Wrapper
	b.Form("post", "").
		Attr("hx-post", endpoint).
		Attr("hx-swap", "none").
		Attr("hx-push-url", "false").
		Script("init send checkFormRules(changed:me as Values)").
		EndBracket()

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Controls
	b.Div()

	if deleteURL := optionMap.GetString("delete"); deleteURL != "" {
		b.Span().Class("float-right", "text-red").Role("button").Attr("hx-get", deleteURL).Attr("hx-push-url", "false").InnerText("Delete").Close()
		b.Space()
	}

	submitLabel := first.String(optionMap.GetString("submit-label"), "Save Changes")
	savingLabel := first.String(optionMap.GetString("saving-label"), "Saving...")
	b.Button().Type("submit").Class("htmx-request-hide primary").InnerText(submitLabel).Close()
	b.Button().Type("button").Class("htmx-request-show primary").Attr("disabled", "true").InnerText(savingLabel).Close()

	if cancelButton := optionMap.GetString("cancel-button"); cancelButton != "hide" {
		cancelLabel := first.String(optionMap.GetString("cancel-label"), "Cancel")
		b.Space()
		b.Button().Type("button").Script("on click trigger closeModal").InnerText(cancelLabel).Close()
		b.Space()
	}

	b.Span().ID("htmx-response-message").Close()

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalForm(response *echo.Response, endpoint string, content string, options ...string) string {
	return WrapModal(response, WrapForm(endpoint, content, options...), options...)
}

// CloseModal sets Response header to close a modal on the client and optionally forward to a new location.
func CloseModal(ctx echo.Context, url string) {

	if url == "" {
		ctx.Response().Header().Set("HX-Trigger", `{"closeModal":"", "refreshPage": ""}`)
	} else {
		ctx.Response().Header().Set("HX-Trigger", `closeModal`)
		ctx.Response().Header().Set("HX-Redirect", url)
	}
}

func RefreshPage(ctx echo.Context) {
	header := ctx.Response().Header()
	header.Set("HX-Trigger", "refreshPage")
	header.Set("HX-Reswap", "none")
}

func TriggerEvent(ctx echo.Context, event string) {
	ctx.Response().Header().Set("HX-Trigger", event)
}

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx *steranko.Context) model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(*model.Authorization); ok {
			return *auth
		}
	}

	return model.NewAuthorization()
}

// parseOptions parses a string of options into a map of key/value pairs
func parseOptions(options ...string) mapof.Any {

	result := mapof.NewAny()

	for _, item := range options {
		head, tail := list.Split(item, ':')
		result.SetString(head, tail)
	}

	return result
}

// replaceActionID replaces the actionID in the URL with the new value
func replaceActionID(path string, newActionID string) string {

	path = strings.TrimPrefix(path, "/")
	parsedPath := list.Head(path, list.DelimiterSlash)

	return "/" + parsedPath + "/" + newActionID
}

type TemplateLike interface {
	Execute(wr io.Writer, data interface{}) error
}

// executeTemplate returns the result of a template execution as a string
func executeTemplate(template TemplateLike, data any) string {

	var buffer bytes.Buffer

	// nolint:errcheck
	if err := template.Execute(&buffer, data); err != nil {
		derp.Report(derp.Wrap(err, "render.executeTemplate", "Error executing template", data))
		return ""
	}

	return buffer.String()
}

// isUserVisible returns TRUE if the currently signed in user is allowed to
// view the provided model.User record.
func isUserVisible(context *steranko.Context, user *model.User) bool {

	authorization := getAuthorization(context)

	// If the user is the domain owner, they can see everything
	if authorization.DomainOwner {
		return true
	}

	// If the user is the same as the one being viewed, they can always see themselves
	if authorization.UserID == user.UserID {
		return true
	}

	// Otherwise, only public users are visible.
	return user.IsPublic
}
