package build

import (
	"context"
	"net/http/httptest"
	"testing"
	"text/template"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestParseBackgroundPath verifies which paths name a background, and the action each one names
func TestParseBackgroundPath(t *testing.T) {

	tests := []struct {
		path   string
		token  string
		action string
		ok     bool
	}{
		{path: "/@ben", token: "@ben", action: "view", ok: true},
		{path: "/@ben/settings", token: "@ben", action: "settings", ok: true},
		{path: "/0123456789abcdef01234567", token: "0123456789abcdef01234567", action: "view", ok: true},
		{path: "/photo/page", token: "photo", action: "page", ok: true},
		{path: "/"},
		{path: ""},
		{path: "/a/b/c"},
		{path: "/a/"},
		{path: "//a"},
	}

	for _, test := range tests {
		token, action, ok := parseBackgroundPath(test.path)
		require.Equal(t, test.ok, ok, test.path)
		require.Equal(t, test.token, token, test.path)
		require.Equal(t, test.action, action, test.path)
	}
}

// TestIsLocalPath verifies that only a path on this server counts as local
func TestIsLocalPath(t *testing.T) {
	require.True(t, isLocalPath("/@ben?tab=1"))
	require.True(t, isLocalPath("/"))
	require.False(t, isLocalPath(""))
	require.False(t, isLocalPath("@ben"))
	require.False(t, isLocalPath("//evil.example/x"))
	require.False(t, isLocalPath(`/\evil.example/x`))
	require.False(t, isLocalPath("https://evil.example/x"))
	require.False(t, isLocalPath("javascript:alert(1)"))
}

// TestIsBackgroundRequest verifies that only a request marked by newBackgroundBuilder counts
func TestIsBackgroundRequest(t *testing.T) {

	request := httptest.NewRequest("GET", "/", nil)
	require.False(t, isBackgroundRequest(request))

	marked := request.WithContext(context.WithValue(request.Context(), backgroundRequestKey{}, true))
	require.True(t, isBackgroundRequest(marked))
}

// TestInsertHead verifies that the modal's head values land before </head>, replacing the page's
// <title> only when they carry one of their own
func TestInsertHead(t *testing.T) {

	page := `<html><head><title>Profile</title><link rel="x"></head><body><title>Not Head</title></body></html>`

	// Nothing to insert leaves the page alone
	require.Equal(t, page, insertHead(page, ""))
	require.Equal(t, page, insertHead(page, "  \n"))

	// Meta tags without a title keep the page's title
	require.Equal(t,
		`<html><head><title>Profile</title><link rel="x"><meta property="og:title" content="Photo"></head><body><title>Not Head</title></body></html>`,
		insertHead(page, `<meta property="og:title" content="Photo">`),
	)

	// A title replaces the page's own, and never one outside <head>
	require.Equal(t,
		`<html><head><link rel="x"><title>Photo</title></head><body><title>Not Head</title></body></html>`,
		insertHead(page, `<title>Photo</title>`),
	)

	// A page without </head> is left alone
	require.Equal(t, "<p>hi</p>", insertHead("<p>hi</p>", "<title>x</title>"))

	// A page whose head has no title just gains one
	require.Equal(t, "<head><title>x</title></head>", insertHead("<head></head>", "<title>x</title>"))
}

// TestMarkReturnURL verifies that the modal's wrapper is given the background's address, escaped
func TestMarkReturnURL(t *testing.T) {

	modal := `<div id="modal" script="install Modal"><div id="modal-underlay"></div></div>`

	require.Equal(t, modal, markReturnURL(modal, ""))
	require.Equal(t,
		`<div id="modal" data-return-url="/@ben?tab=1&amp;x=&#34;y&#34;" script="install Modal"><div id="modal-underlay"></div></div>`,
		markReturnURL(modal, `/@ben?tab=1&x="y"`),
	)
}

// TestStepAsModal_ReplaceURL verifies that replace-url is evaluated against the builder, and that
// anything but a local path is refused
func TestStepAsModal_ReplaceURL(t *testing.T) {

	parse := func(replaceURL string) StepAsModal {
		parsed, err := step.NewAsModal(mapof.Any{"replace-url": replaceURL}, template.FuncMap{})
		require.NoError(t, err)
		return StepAsModal(parsed)
	}

	require.Equal(t, "/new", parse("/{{if .IsNew}}new{{else}}old{{end}}").replaceURL(asModalBuilder{isNew: true}))
	require.Equal(t, "", parse("https://evil.example/{{.IsNew}}").replaceURL(asModalBuilder{}))
	require.Equal(t, "", parse("//evil.example").replaceURL(asModalBuilder{}))
	require.Equal(t, "", StepAsModal{}.replaceURL(asModalBuilder{}))
}

// TestCommon_IsBackground verifies that a builder reports a request drawn behind a modal
func TestCommon_IsBackground(t *testing.T) {

	request := httptest.NewRequest("GET", "/", nil)
	require.False(t, Common{_request: request}.IsBackground())

	marked := request.WithContext(context.WithValue(request.Context(), backgroundRequestKey{}, true))
	require.True(t, Common{_request: marked}.IsBackground())
}
