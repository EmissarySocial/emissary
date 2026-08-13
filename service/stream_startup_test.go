package service

import (
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/davidscottmills/goeditorjs"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestStream_Startup_DefaultThemeStreams is the regression test for the first-run
// onboarding loop: completing the /startup wizard saved the theme but created zero
// streams, so /home had no "home" stream and 307-redirected back to /startup forever.
//
// Two stacked defects had to be fixed for a fresh domain to be initializable:
//
//  1. Stream.Startup fed each theme.StartupStreams entry -- which carries the article
//     body under a whole "content" model.Content object -- straight through
//     streamSchema.SetAll, and rosetta's object-Set cannot assign a whole object in one
//     call.  Every startup stream failed at that first step and was silently skipped.
//
//  2. Once SetAll was fixed, the pre-Save streamSchema.Validate call rejected the streams,
//     because Validate treats ANY rewrite as failure and the "html" format sanitizer
//     always rewrites freshly-rendered article HTML.  Save (which Normalizes rather than
//     Validates) is the correct gate.
//
// This test loads the REAL shipping default theme (theme.hjson + its content/*.{json,md}
// files, injected exactly as the Theme service does) so it can't drift from what a fresh
// install actually feeds Startup.  It drives the extracted newStartupStream builder and
// then Normalizes -- the same operation Save performs before persisting -- proving each
// startup stream, most importantly "home", is both buildable and persistable.
func TestStream_Startup_DefaultThemeStreams(t *testing.T) {

	const themeDir = "../_embed/templates/theme-default"

	// Load the real default theme definition
	definition, err := os.ReadFile(themeDir + "/theme.hjson")
	require.NoError(t, err, "unable to read default theme definition")

	theme := model.NewTheme("default", nil)
	require.NoError(t, hjson.Unmarshal(definition, &theme), "unable to parse default theme")
	require.NotEmpty(t, theme.StartupStreams, "default theme must define startup streams")

	// Inject default content the same way service.Theme does on load, using the real
	// Content service (with the same editorJS block handlers as production) so the
	// "content" key holds a genuine, fully-rendered model.Content object.
	engine := goeditorjs.NewHTMLEngine()
	engine.RegisterBlockHandlers(
		&goeditorjs.HeaderHandler{},
		&goeditorjs.ParagraphHandler{},
		&goeditorjs.ListHandler{},
		&goeditorjs.ImageHandler{},
		&goeditorjs.RawHTMLHandler{},
	)
	contentService := NewContent(engine)

	themeService := NewTheme(nil, &contentService, nil)
	themeService.setStartupContent(&theme, os.DirFS(themeDir+"/content"))

	// newStartupStream + Schema().Normalize are both dependency-free
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

		// End-to-end proof: the built Stream must survive the same normalization that Save
		// applies before persisting.  This is the step that failed once SetAll was fixed --
		// Normalize rewrites the sanitized HTML in place rather than rejecting it, so a
		// fresh domain can actually be initialized.
		_, err = streamService.Schema().Normalize(&stream)
		require.NoError(t, err, "startup stream %q must normalize for Save", token)

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

	// The built Stream must also survive Save's normalization gate.
	_, err = streamService.Schema().Normalize(&stream)
	require.NoError(t, err)
}

// newStartupTokensTheme builds a Theme whose startup Streams carry the provided tokens, in order.
func newStartupTokensTheme(tokens ...string) model.Theme {

	theme := model.NewTheme("test", nil)

	for index, token := range tokens {
		theme.StartupStreams = append(theme.StartupStreams, mapof.Any{
			"templateId": "article-markdown",
			"token":      token,
			"label":      "Page " + token,
			"rank":       index + 1,
		})
	}

	return theme
}

// selectedTokens reduces a selection back to the tokens it contains, so the assertions below
// read as the list a user checked rather than as a pile of maps.
func selectedTokens(selection []mapof.Any) []string {

	result := make([]string, 0, len(selection))

	for _, data := range selection {
		result = append(result, data.GetString("token"))
	}

	return result
}

// TestStream_selectStartupStreams covers the whole selection matrix: which Streams a request
// creates is decided here, and every class of input a form POST can produce -- nothing checked,
// a subset, everything, duplicates, and values the Theme never offered -- has to land somewhere
// predictable.
func TestStream_selectStartupStreams(t *testing.T) {

	theme := newStartupTokensTheme("home", "about", "join-the-team")

	// selectsTokens asserts that requesting `tokens` selects exactly `expected`, in Theme order.
	selectsTokens := func(name string, tokens sliceof.String, expected []string) {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, selectedTokens(selectStartupStreams(&theme, tokens)))
		})
	}

	selectsTokens("nil selects nothing", nil, []string{})
	selectsTokens("empty selects nothing", sliceof.String{}, []string{})
	selectsTokens("one token selects one Stream", sliceof.String{"about"}, []string{"about"})
	selectsTokens("a subset selects only that subset", sliceof.String{"home", "join-the-team"}, []string{"home", "join-the-team"})
	selectsTokens("every token selects every Stream", sliceof.String{"home", "about", "join-the-team"}, []string{"home", "about", "join-the-team"})

	// The Theme -- not the request -- decides the order and the multiplicity of the result, so a
	// reversed or repeated request cannot reorder or duplicate the Streams that get created.
	selectsTokens("request order does not reorder the result", sliceof.String{"join-the-team", "home"}, []string{"home", "join-the-team"})
	selectsTokens("a repeated token creates one Stream", sliceof.String{"about", "about"}, []string{"about"})

	// RULE: The Theme is the authority on what CAN be created.  A token that it does not define
	// is dropped, no matter what the browser posted.
	selectsTokens("an unknown token selects nothing", sliceof.String{"../../etc/passwd"}, []string{})
	selectsTokens("an empty token selects nothing", sliceof.String{""}, []string{})
	selectsTokens("unknown tokens do not disturb known ones", sliceof.String{"nope", "home", ""}, []string{"home"})

	// A Theme with no startup Streams has nothing to offer, whatever is requested.
	t.Run("an empty Theme selects nothing", func(t *testing.T) {
		empty := newStartupTokensTheme()
		require.Equal(t, []string{}, selectedTokens(selectStartupStreams(&empty, sliceof.String{"home"})))
	})
}

// TestStream_Startup_EmptySelectionSkipsDatabase pins the ordering inside Startup: an empty
// selection must return BEFORE any database access.  The nil session is the proof -- a Startup
// that counted Streams first would panic on it instead of returning cleanly.
func TestStream_Startup_EmptySelectionSkipsDatabase(t *testing.T) {

	theme := newStartupTokensTheme("home", "about")
	streamService := &Stream{}

	require.NoError(t, streamService.Startup(nil, &theme, nil))
	require.NoError(t, streamService.Startup(nil, &theme, sliceof.String{}))

	// A token the Theme does not define selects nothing, and so must not reach the database either.
	require.NoError(t, streamService.Startup(nil, &theme, sliceof.String{"not-a-real-token"}))
}

// TestTheme_StartupStreamTokens covers the "everything the Theme defines" list that callers with
// no selection of their own pass to Startup.  Feeding it back into the selection must reproduce
// the Theme's whole startup list -- that round trip is what preserves the old create-everything
// behavior for the /startup POST handler.
func TestTheme_StartupStreamTokens(t *testing.T) {

	theme := newStartupTokensTheme("home", "about", "join-the-team")

	require.Equal(t, sliceof.String{"home", "about", "join-the-team"}, theme.StartupStreamTokens())
	require.Equal(t, []string{"home", "about", "join-the-team"}, selectedTokens(selectStartupStreams(&theme, theme.StartupStreamTokens())))

	// An empty Theme yields an empty (not nil) list, which selects nothing.
	empty := newStartupTokensTheme()
	require.Equal(t, sliceof.String{}, empty.StartupStreamTokens())
}
