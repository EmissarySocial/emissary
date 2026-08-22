package service

import (
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// RULE: A Theme's "startupStreams" are all-or-nothing.  Stream.Startup takes no selection
// argument, so a caller creates every Stream that the Theme defines, or none of them.  The
// tests below pin both halves of that rule: the API cannot be asked for a subset, and a
// non-empty Theme cannot be talked out of reaching the database.

// TestStream_Startup_TakesNoSelection is the compile-time gate on the rule above.  Startup used
// to accept a `tokens sliceof.String` of user-chosen Streams; re-adding any such parameter --
// or any other way to narrow what a Theme installs -- breaks this assignment.  That matters
// because the selection previously arrived straight off a form POST, which made "which content
// does a fresh domain start with" a browser-controlled decision.
func TestStream_Startup_TakesNoSelection(t *testing.T) {

	streamService := &Stream{}

	var startup func(data.Session, *model.Theme) error = streamService.Startup

	require.NotNil(t, startup)
}

// TestStream_Startup_EmptyThemeSkipsDatabase pins the ordering inside Startup: a Theme with no
// startup Streams must return BEFORE any database access.  The nil session is the proof -- a
// Startup that counted Streams first would panic on it instead of returning cleanly.
func TestStream_Startup_EmptyThemeSkipsDatabase(t *testing.T) {

	streamService := &Stream{}

	empty := newStartupTokensTheme()
	require.Empty(t, empty.StartupStreams)

	require.NoError(t, streamService.Startup(nil, &empty))
}

// TestStream_Startup_NonEmptyThemeReachesDatabase is the inverse, and it is the test that would
// have failed under the old design.  Only the Theme's own list can now short-circuit Startup, so
// a Theme that defines Streams MUST fall through to the Count/Save work -- the nil session turns
// "reached the database" into an observable panic.  If a filter ever creeps back in and drops
// every Stream, this test goes green-to-red rather than silently seeding an empty domain.
func TestStream_Startup_NonEmptyThemeReachesDatabase(t *testing.T) {

	streamService := &Stream{}

	theme := newStartupTokensTheme("home", "about")

	require.Panics(t, func() {
		_ = streamService.Startup(nil, &theme)
	}, "a Theme with startup Streams must reach the database")
}

// TestStream_Startup_DefaultThemeStreams is the regression test for the first-run onboarding
// loop: completing the /startup wizard saved the theme but created zero streams, so /home had no
// "home" stream and 307-redirected back to /startup forever.
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
//
// Because there is no selection step any more, EVERY entry here is one that a fresh domain
// will actually create, which is why the loop asserts on all of them rather than on a sample.
func TestStream_Startup_DefaultThemeStreams(t *testing.T) {

	theme := loadDefaultTheme(t)

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

// TestStream_Startup_DefaultThemeTokensAreUnique guards the shipping theme itself.  Startup now
// installs the whole list unconditionally, so two entries sharing a "token" would race for the
// same URL on every fresh install instead of being deduplicated by a selection step.
func TestStream_Startup_DefaultThemeTokensAreUnique(t *testing.T) {

	theme := loadDefaultTheme(t)

	seen := make(map[string]bool, len(theme.StartupStreams))

	for _, data := range theme.StartupStreams {

		token := data.GetString("token")

		require.NotEmpty(t, token, "every startup stream must define a token")
		require.False(t, seen[token], "duplicate startup stream token %q", token)

		seen[token] = true
	}
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

// TestStream_newStartupStream_DoesNotMutateTheme covers a hazard that the all-or-nothing design
// makes load-bearing: Themes are parsed once and shared across every request, and newStartupStream
// has to strip the "content" key before calling SetAll.  Copying the map (rather than deleting the
// key) is what keeps the second domain to start up from getting content-less Streams.
func TestStream_newStartupStream_DoesNotMutateTheme(t *testing.T) {

	streamService := &Stream{}

	data := mapof.Any{
		"templateId": "article-editorjs",
		"token":      "home",
		"content": model.Content{
			Format: model.ContentFormatEditorJS,
			HTML:   `<p>hi</p>`,
		},
	}

	// Build twice from the SAME map, exactly as two domains sharing one cached Theme would.
	first, err := streamService.newStartupStream(data)
	require.NoError(t, err)

	second, err := streamService.newStartupStream(data)
	require.NoError(t, err)

	require.Contains(t, data, "content", "the shared Theme map must still carry its content")
	require.Equal(t, first.Content.HTML, second.Content.HTML)
	require.Equal(t, `<p>hi</p>`, second.Content.HTML)
}

// TestStream_newStartupStream_RejectsUnknownKeys pins the failure mode for a malformed Theme.
// A key that the Stream schema does not define is an error, not a silent skip, so a typo in a
// theme.hjson stops the whole Startup instead of quietly shipping a half-configured page.
func TestStream_newStartupStream_RejectsUnknownKeys(t *testing.T) {

	streamService := &Stream{}

	_, err := streamService.newStartupStream(mapof.Any{
		"token":    "home",
		"labelTyp": "Welcome!", // a plausible typo for "label"
	})

	require.Error(t, err, "an unknown key must fail loudly")
}

// TestStream_newStartupStream_IgnoresNonContentValue documents the one key that fails QUIETLY.
// "content" bypasses the schema entirely, so a Theme that supplies a string (or anything else
// that is not a model.Content) gets an empty body rather than an error.  This is the behavior
// as built; the test exists so that a change to it is a deliberate one.
func TestStream_newStartupStream_IgnoresNonContentValue(t *testing.T) {

	streamService := &Stream{}

	stream, err := streamService.newStartupStream(mapof.Any{
		"token":   "home",
		"content": "<p>this is not a model.Content</p>",
	})

	require.NoError(t, err)
	require.Empty(t, stream.Content.HTML)
	require.Empty(t, stream.Content.Raw)
}

// TestStream_newStartupStream_ForcesPublished pins the one field that the Theme cannot control.
// Startup Streams exist to be visible on a brand-new domain, so a "publishDate" in the Theme --
// which the schema WILL happily set -- must still be overwritten with zero.
func TestStream_newStartupStream_ForcesPublished(t *testing.T) {

	streamService := &Stream{}

	stream, err := streamService.newStartupStream(mapof.Any{
		"token":       "home",
		"publishDate": 4102444800, // far-future: unpublished if it survived
	})

	require.NoError(t, err)
	require.Zero(t, stream.PublishDate, "a Theme must not be able to hold a startup Stream back")
}

// loadDefaultTheme parses the REAL shipping default theme and injects its default content the
// same way service.Theme does on load, using the real Content service (with the same editorJS
// block handlers as production) so the "content" key holds a genuine, fully-rendered
// model.Content object.  Tests read the theme off disk rather than fixture it so they cannot
// drift from what a fresh install actually feeds Startup.
func loadDefaultTheme(t *testing.T) model.Theme {

	t.Helper()

	const themeDir = "../_embed/templates/theme-default"

	definition, err := os.ReadFile(themeDir + "/theme.hjson")
	require.NoError(t, err, "unable to read default theme definition")

	theme := model.NewTheme("default", nil)
	require.NoError(t, hjson.Unmarshal(definition, &theme), "unable to parse default theme")
	require.NotEmpty(t, theme.StartupStreams, "default theme must define startup streams")

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

	return theme
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
