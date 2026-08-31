package service

import (
	"bytes"
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Redirect Template
 *
 * A Redirect forwards visitors to an off-site URL, and it is the
 * only shipped Template whose whole job is a cross-origin hop.
 * The mechanism that makes that work for both an <a href> and an
 * hx-get lives in build/navigation.go, which the "redirect-to"
 * step reaches through navigateContent -- so this Template needs
 * no branch of its own, and these tests pin that it does not
 * grow one back.
 ******************************************/

// redirectTemplate parses the shipped redirect definition
func redirectTemplate(t *testing.T) model.Template {

	t.Helper()

	definition, err := os.ReadFile("../_embed/templates/stream-redirect/template.hjson")
	require.NoError(t, err)

	result := model.NewTemplate("redirect", nil)
	require.NoError(t, hjson.Unmarshal(definition, &result))

	return result
}

// redirectBuilderStub stands in for a build.Stream when executing the Template's own
// text/template expressions.  Only the method those expressions call is needed, because
// text/template resolves by name.
type redirectBuilderStub struct{}

// Data returns the stub's target URL for the "url" key, mirroring build.Stream.Data
func (stub redirectBuilderStub) Data(value string) any {

	if value == "url" {
		return "https://instagram.com"
	}

	return nil
}

// TestRedirectTemplate_ViewIsOneRedirect confirms that "view" is a single "redirect-to"
// covering GET, and that its target is the configured URL.  The htmx-versus-browser
// decision is made in Go, so a branch here would be a sign that it regressed.
func TestRedirectTemplate_ViewIsOneRedirect(t *testing.T) {

	action, exists := redirectTemplate(t).Actions["view"]
	require.True(t, exists, "template has no \"view\" action")
	require.Len(t, action.Steps, 1, "\"view\" must be exactly one step")

	redirectTo, ok := action.Steps[0].(modelStep.RedirectTo)
	require.True(t, ok, "\"view\" must be a \"redirect-to\", not a %s", action.Steps[0].Name())
	require.Contains(t, []string{"get", "both"}, redirectTo.Method, "redirect-to must run on GET")

	var url bytes.Buffer
	require.NoError(t, redirectTo.URL.Execute(&url, redirectBuilderStub{}))
	require.Equal(t, "https://instagram.com", url.String())
}

// TestRedirectTemplate_AnonymousViewRequiresPublished confirms that the public can only
// follow a Redirect that has been published; "view" grants the roles unconditionally.
func TestRedirectTemplate_AnonymousViewRequiresPublished(t *testing.T) {

	action, exists := redirectTemplate(t).Actions["view"]
	require.True(t, exists)

	require.NotContains(t, action.Roles, "anonymous")
	require.Contains(t, action.StateRoles["published"], "anonymous")
}
