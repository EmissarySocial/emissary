package build

import (
	"bytes"
	"io"
	"mime/multipart"
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

// WrapInlineSuccess sends a confirmation message to the #htmx-response-message element
func WrapInlineSuccess(response http.ResponseWriter, message any) error {

	response.Header().Set("HX-Reswap", "innerHTML")
	response.Header().Set("HX-Retarget", "#htmx-response-message")
	response.WriteHeader(http.StatusOK)

	_, err := response.Write([]byte(`<span class="green">` + convert.String(message) + `</span>`))
	return derp.Wrap(err, "build.WrapInlineSuccess", "Error writing response", message)
}

// WrapInlineError sends an error message to the #htmx-response-message element
func WrapInlineError(response http.ResponseWriter, err error) error {

	derp.Report(err)

	response.Header().Set("HX-Reswap", "innerHTML")
	response.Header().Set("HX-Retarget", "#htmx-response-message")
	response.WriteHeader(http.StatusOK)

	if _, writeError := response.Write([]byte(`<span class="text-red">` + derp.Message(err) + `</span>`)); writeError != nil {
		return derp.Wrap(writeError, "build.WrapInlineError", "Error writing response", err)
	}

	return nil
}

func WrapModal(response http.ResponseWriter, content string, options ...string) string {

	// These three headers make it a modal
	header := response.Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Reswap", "innerHTML")
	header.Set("HX-Push-Url", "false")

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

func WrapModalWithCloseButton(response http.ResponseWriter, content string, options ...string) string {
	b := html.New()

	b.Div()
	b.Button().Script("on click trigger closeModal").InnerText("Close Window")

	return WrapModal(response, content+b.String())
}

func WrapTooltip(response http.ResponseWriter, content string) string {

	// These headers make it a modal
	header := response.Header()
	header.Set("HX-Reswap", "beforeend")
	header.Set("HX-Push-Url", "false")

	b := html.New()

	b.Span().ID("tooltip").Script("install tooltip").EndBracket()
	b.WriteString(content)
	b.CloseAll()

	return b.String()
}

func WrapForm(endpoint string, content string, encoding string, options ...string) string {

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
		Attr("hx-encoding", encoding).
		Attr("hx-trigger", "submit").
		Script("init send checkFormRules(changed:me as Values)").
		EndBracket()

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Controls
	b.Div()

	submitLabel := first.String(optionMap.GetString("submit-label"), "Save Changes")

	b.Div().Class("flex-row")
	b.Div().Class("flex-grow")
	{
		b.Button().Type("submit").ID("inline-save-button").Class("primary").TabIndex("0").Script("install SaveButton").InnerText(submitLabel).Close()

		if cancelButton := optionMap.GetString("cancel-button"); cancelButton != "hide" {
			cancelLabel := first.String(optionMap.GetString("cancel-label"), "Cancel")
			b.Space()
			b.Button().Type("button").Script("on click trigger closeModal").TabIndex("0").InnerText(cancelLabel).Close()
			b.Space()
		}

		b.Span().ID("htmx-response-message").Close()
	}
	b.Close()

	if deleteURL := optionMap.GetString("delete"); deleteURL != "" {
		deleteLabel := first.String(optionMap.GetString("delete-label"), "Delete")
		b.Div()
		b.Span().Class("text-red").Role("button").Attr("hx-get", deleteURL).Attr("hx-push-url", "false").InnerText(deleteLabel).Close()
		b.Close()
	}

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalForm(response http.ResponseWriter, endpoint string, content string, encoding string, options ...string) string {
	return WrapModal(response, WrapForm(endpoint, content, encoding, options...), options...)
}

// CloseModal sets Response header to close a modal on the client and optionally forward to a new location.
func CloseModal(ctx echo.Context) {
	ctx.Response().Header().Set("HX-Trigger", `{"closeModal":"", "refreshPage": ""}`)
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
func getAuthorization(st *steranko.Steranko, request *http.Request) model.Authorization {

	if claims, err := st.GetAuthorization(request); err == nil {

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

	if err := template.Execute(&buffer, data); err != nil {
		derp.Report(derp.Wrap(err, "build.execute", "Error executing template", data))
		return ""
	}

	return buffer.String()
}

// AsHTML collects the logic to build complete vs. partial HTML pages.
func AsHTML(factory Factory, ctx echo.Context, b Builder, actionMethod ActionMethod) error {

	const location = "build.AsHTML"
	var partialPage bytes.Buffer

	// Execute the action pipeline
	pipeline := Pipeline(b.action().Steps)

	status := pipeline.Execute(factory, b, &partialPage, actionMethod)

	if status.Error != nil {
		return derp.Wrap(status.Error, location, "Error executing action pipeline")
	}

	// Copy status values into the Response...
	status.Apply(ctx.Response())

	// Partial page requests can be completed here.
	if b.IsPartialRequest() || status.FullPage {
		if err := ctx.HTML(status.GetStatusCode(), partialPage.String()); err != nil {
			return derp.Wrap(err, location, "Error building partial-page content", status.GetStatusCode())
		}

		return nil
	}

	// Full Page requests require the theme service to wrap the built content
	htmlTemplate := factory.Domain().Theme().HTMLTemplate
	b.SetContent(partialPage.String())
	var fullPage bytes.Buffer

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", b); err != nil {
		return derp.Wrap(err, location, "Error building full-page content")
	}

	return ctx.HTML(http.StatusOK, fullPage.String())
}

// isUserVisible returns TRUE if the currently signed in user is allowed to
// view the provided model.User record.
func isUserVisible(authorization *model.Authorization, user *model.User) bool {

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

// bind replicates the echo.Context.Bind() function without
// using an echo.Context
func bind(request *http.Request, result any) error {
	e := echo.New()
	ctx := e.NewContext(request, nil)
	return ctx.Bind(result)
}

// bindBody replicates the echo.Context.Bind() function without
// requiring an external echo.Context.  This function ONLY binds the
// request body, and ignores other parameters (e.g. path, query, and headers).
func bindBody(request *http.Request, result any) error {
	e := echo.New()
	ctx := e.NewContext(request, nil)
	binder := echo.DefaultBinder{}
	return binder.BindBody(ctx, result)
}

// multipartForm replicates the echo.Context.MultipartForm() function without
// using an echo.Context
func multipartForm(request *http.Request) (*multipart.Form, error) {

	if err := request.ParseMultipartForm(32 << 20); err != nil {
		return nil, derp.Wrap(err, "build.multipartForm", "Error parsing multipart form")
	}

	return request.MultipartForm, nil
}

// redirect replicates the echo.Context.Redirect() function without
// using an echo.Context
func redirect(response http.ResponseWriter, statusCode int, location string) error {
	response.Header().Add("Location", location)
	response.WriteHeader(statusCode)
	return nil
}

// getTemplate returns the model.Template from a Builder, if it exists
func getTemplate(builder Builder) (model.Template, bool) {

	if templateGetter, ok := builder.(templateGetter); ok {
		return templateGetter.template(), true
	}

	return model.Template{}, false
}
