package render

import (
	"html/template"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// mustTemplate guarantees that a value is a template.Template, or else it is replaced with an empty template.
func mustTemplate(data interface{}) *template.Template {
	if t, ok := data.(*template.Template); ok {
		return t
	}

	return template.New("mising")
}

// isPartialPageRequest returns TRUE if this request was made by `hx-get`
func isPartialPageRequest(ctx echo.Context) bool {
	return (ctx.Request().Header.Get("HX-Request") != "")
}

// getActionID returns the :action token from the Request (or a default)
func getActionID(ctx *steranko.Context) string {

	if ctx.Request().Method == http.MethodDelete {
		return "delete"
	}

	if actionID := ctx.Param("action"); actionID != "" {
		return actionID
	}

	return "view"
}

// getAction locates and populates the action.Action for a specific template and actionID
func getAction(templateService *service.Template, stream *model.Stream, authorization *model.Authorization, actionID string) (model.Action, error) {

	// Try to find the action based on the stream and actionID
	result, err := templateService.Action(stream.TemplateID, actionID)

	if err != nil {
		return model.Action{}, derp.Wrap(err, "ghost.render.NewAction", "Could not create action", stream, actionID)
	}

	// Enforce user permissions here.
	if !result.UserCan(stream, authorization) {
		return model.Action{}, derp.New(derp.CodeForbiddenError, "ghost.render.NewAction", "Forbidden", stream, authorization)
	}

	return result, nil
}

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx *steranko.Context) *model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(*model.Authorization); ok {
			return auth
		}
	}

	result := model.NewAuthorization()
	return &result
}

func WrapModalForm(renderer *Renderer, content string) string {

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal")
	b.Div().Class("modal-underlay").Script("on click send closeModal to #modal").Close()
	b.Div().Class("modal-content")

	// Form Wrapper
	b.Form("post", "").Attr("hx-post", renderer.URL()).EndBracket()

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Controls
	b.Div()
	b.Input("submit", "").Class("primary").Value("Save Changes").Close()
	b.WriteString("&nbsp;")
	b.Span().Class("button").Script("on click trigger closeModal").InnerHTML("Cancel").Close()

	// Done
	b.CloseAll()

	return b.String()
}
