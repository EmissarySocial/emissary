package service

import (
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestStream_Startup_DefaultThemeStreams is the regression test for the first-run
// onboarding loop: completing the /startup wizard saved the theme but created zero
// streams, so /home had no "home" stream and 307-redirected back to /startup forever.
//
// The root cause was that Stream.Startup fed each theme.StartupStreams entry -- which
// carries the article body under a whole "content" model.Content object -- straight
// through streamSchema.SetAll, and rosetta's object-Set cannot assign a whole object in
// one call.  Every startup stream failed at that first step and was silently skipped.
//
// This test loads the REAL shipping default theme (theme.hjson + its content/*.{json,md}
// files, injected exactly as the Theme service does) so it can't drift from what a fresh
// install actually feeds Startup, then drives the extracted newStartupStream seam and
// asserts every startup stream -- most importantly "home" -- builds without error.
func TestStream_Startup_DefaultThemeStreams(t *testing.T) {

	const themeDir = "../_embed/templates/theme-default"

	// Load the real default theme definition
	definition, err := os.ReadFile(themeDir + "/theme.hjson")
	require.NoError(t, err, "unable to read default theme definition")

	theme := model.NewTheme("default", nil)
	require.NoError(t, hjson.Unmarshal(definition, &theme), "unable to parse default theme")
	require.NotEmpty(t, theme.StartupStreams, "default theme must define startup streams")

	// Inject default content the same way service.Theme does on load, using the real
	// Content service so the "content" key holds a genuine model.Content object.
	engine := goeditorjs.NewHTMLEngine()
	engine.RegisterBlockHandlers(&goeditorjs.HeaderHandler{}, &goeditorjs.ParagraphHandler{})
	contentService := NewContent(engine)

	themeService := NewTheme(nil, &contentService, nil)
	themeService.setStartupContent(&theme, os.DirFS(themeDir+"/content"))

	// newStartupStream needs only the (dependency-free) Stream schema
	streamService := &Stream{}

	foundHome := false

	for _, data := range theme.StartupStreams {

		token := data.GetString("token")

		stream, err := streamService.newStartupStream(data)
		require.NoError(t, err, "startup stream %q must build without error", token)

		// The scalar fields must survive schema application
		require.Equal(t, token, stream.Token)
		require.Equal(t, data.GetString("templateId"), stream.TemplateID)
		require.Equal(t, data.GetString("label"), stream.Label)

		// Startup streams must be published (PublishDate reset to 0)
		require.Zero(t, stream.PublishDate, "startup stream %q must be published", token)

		// The whole-object content must survive as well
		if content, ok := data["content"].(model.Content); ok {
			require.Equal(t, content.Format, stream.Content.Format, "content must survive for %q", token)
			require.Equal(t, content.Raw, stream.Content.Raw, "content must survive for %q", token)
		}

		if token == "home" {
			foundHome = true
		}
	}

	// The "home" stream is the one whose absence caused the redirect loop
	require.True(t, foundHome, "default theme must yield a 'home' startup stream")
}

// TestStream_newStartupStream_ExcludesContentObject pins the exact contract that broke:
// a whole "content" object cannot be assigned through the stream schema's SetAll, so
// newStartupStream must exclude it from SetAll and apply it directly.  The naive SetAll
// assertion documents WHY the special-casing exists -- if a future rosetta makes
// whole-object Set work, this test surfaces the change instead of hiding it.
func TestStream_newStartupStream_ExcludesContentObject(t *testing.T) {

	streamService := &Stream{}

	data := mapof.Any{
		"templateId": "article-editorjs",
		"token":      "home",
		"label":      "Welcome!",
		"rank":       1,
		"content": model.Content{
			Format: model.ContentFormatEditorJS,
			Raw:    `{"blocks":[]}`,
			HTML:   `<p>hi</p>`,
		},
	}

	// The original failure: feeding the whole map (content included) through SetAll errors.
	naive := model.NewStream()
	require.Error(t, streamService.Schema().SetAll(&naive, data),
		"a whole content object must not be assignable through SetAll")

	// newStartupStream handles the content key specially and succeeds.
	stream, err := streamService.newStartupStream(data)
	require.NoError(t, err)

	require.Equal(t, "home", stream.Token)
	require.Equal(t, "Welcome!", stream.Label)
	require.Equal(t, "article-editorjs", stream.TemplateID)
	require.Equal(t, model.ContentFormatEditorJS, stream.Content.Format)
	require.Equal(t, `<p>hi</p>`, stream.Content.HTML)
	require.Zero(t, stream.PublishDate)
}
