package model

import (
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// TestNewAttachmentRules confirms that new rules are empty but non-nil
func TestNewAttachmentRules(t *testing.T) {

	rules := NewAttachmentRules()

	require.NotNil(t, rules.Extensions, "must be non-nil so schema writes do not panic")
	require.Equal(t, 0, len(rules.Extensions))
	require.Equal(t, 0, rules.Width)
	require.Equal(t, 0, rules.Height)
	require.Equal(t, 0, rules.Bitrate)
}

// fileSpecFor is a shorthand for applying rules to a request URL
func fileSpecFor(t *testing.T, rules AttachmentRules, rawURL string, originalExtension string) (string, int, int, int) {

	t.Helper()

	address, err := url.Parse(rawURL)
	require.Nil(t, err, "test URL must parse: %q", rawURL)

	filespec := rules.FileSpec(address, originalExtension)

	return filespec.Extension, filespec.Width, filespec.Height, filespec.Bitrate
}

// TestAttachmentRules_FileSpec_Defaults confirms the extensions offered when a Template defines none
func TestAttachmentRules_FileSpec_Defaults(t *testing.T) {

	rules := NewAttachmentRules()

	table := []struct {
		originalExtension string
		requested         string
		expected          string
		reason            string
	}{
		// Images default to WebP, but the other image formats stay available
		{".png", "/file.webp", ".webp", "webp is the image default"},
		{".png", "/file.png", ".png", "png is allowed"},
		{".png", "/file.gif", ".gif", "gif is allowed"},
		{".png", "/file.jpeg", ".jpeg", "jpeg is allowed"},
		{".png", "/file.jpg", ".jpeg", "jpg is folded into jpeg"},
		{".png", "/file.JPG", ".webp", "the fold is case-sensitive, so JPG falls back"},
		{".png", "/file.svg", ".webp", "svg is not an allowed image output"},
		{".png", "/file.html", ".webp", "html is not an allowed image output"},
		{".png", "/file", ".webp", "no extension at all falls back"},

		// The original's *category* drives the defaults, not its exact type
		{".jpeg", "/file.png", ".png", "jpeg original, png output"},
		{".gif", "/file.webp", ".webp", "gif original, webp output"},

		// RULE: An unrecognized original has no default list, so whatever the URL
		// asks for is passed straight through.
		{".pdf", "/file.pdf", ".pdf", "documents have no conversion list"},
		{".pdf", "/file.exe", ".exe", "...so the request is not filtered"},
		{"", "/file.png", ".png", "no original extension, no filtering"},
		{"", "/file", ".", "nothing to go on at all"},
	}

	for _, row := range table {
		extension, _, _, _ := fileSpecFor(t, rules, row.requested, row.originalExtension)
		require.Equal(t, row.expected, extension,
			"original=%q requested=%q (%s)", row.originalExtension, row.requested, row.reason)
	}
}

// TestAttachmentRules_FileSpec_AudioVideoDefaults confirms the audio and video default lists
func TestAttachmentRules_FileSpec_AudioVideoDefaults(t *testing.T) {

	rules := NewAttachmentRules()

	// These extensions are read from the host's mime database, so the category is
	// asserted through the resulting default list rather than the type string itself.
	audio := []string{"mp3", "opus", "aac", "ogg", "flac"}
	video := []string{"mp4", "webm", "ogv", "webp"}

	for _, extension := range audio {
		result, _, _, _ := fileSpecFor(t, rules, "/file."+extension, ".mp3")
		require.Equal(t, "."+extension, result, "audio must allow %q", extension)
	}

	for _, extension := range video {
		result, _, _, _ := fileSpecFor(t, rules, "/file."+extension, ".mp4")
		require.Equal(t, "."+extension, result, "video must allow %q", extension)
	}

	// The first entry in each list is the fallback for anything unrecognized
	result, _, _, _ := fileSpecFor(t, rules, "/file.wav", ".mp3")
	require.Equal(t, ".mp3", result, "audio falls back to mp3")

	result, _, _, _ = fileSpecFor(t, rules, "/file.avi", ".mp4")
	require.Equal(t, ".mp4", result, "video falls back to mp4")

	// Categories do not bleed into each other
	result, _, _, _ = fileSpecFor(t, rules, "/file.mp3", ".mp4")
	require.Equal(t, ".mp4", result, "an audio extension is not a valid video output")
}

// TestAttachmentRules_FileSpec_Allowlist confirms that a Template's extension list is enforced
func TestAttachmentRules_FileSpec_Allowlist(t *testing.T) {

	rules := NewAttachmentRules()
	rules.Extensions = sliceof.String{"webp", "png"}

	table := []struct {
		requested string
		expected  string
		reason    string
	}{
		{"/file.webp", ".webp", "allowed"},
		{"/file.png", ".png", "allowed"},
		{"/file.gif", ".webp", "not allowed, falls back to the first entry"},
		{"/file.jpg", ".webp", "jpg becomes jpeg, which is not allowed"},
		{"/file.html", ".webp", "an executable extension can never survive the list"},
		{"/file", ".webp", "no extension, first entry wins"},
	}

	for _, row := range table {
		extension, _, _, _ := fileSpecFor(t, rules, row.requested, ".png")
		require.Equal(t, row.expected, extension, "requested=%q (%s)", row.requested, row.reason)
	}

	// RULE: An explicit list overrides the category defaults entirely, even when the
	// list contains an extension that the category would never have offered.
	rules.Extensions = sliceof.String{"txt"}
	extension, _, _, _ := fileSpecFor(t, rules, "/file.png", ".png")
	require.Equal(t, ".txt", extension)
}

// TestAttachmentRules_FileSpec_Dimensions confirms how fixed rules override query parameters
func TestAttachmentRules_FileSpec_Dimensions(t *testing.T) {

	table := []struct {
		ruleWidth  int
		ruleHeight int
		requested  string
		width      int
		height     int
		reason     string
	}{
		// With no rules, the caller decides
		{0, 0, "/f.png?width=100&height=50", 100, 50, "caller sets both"},
		{0, 0, "/f.png", 0, 0, "caller sets neither"},
		{0, 0, "/f.png?width=100", 100, 0, "caller sets width only"},

		// RULE: A fixed width pins the width *and* forbids the caller from setting a
		// height, so the image cannot be stretched into a different shape.
		{300, 0, "/f.png?width=100&height=50", 300, 0, "fixed width discards the requested height"},
		{300, 0, "/f.png", 300, 0, "fixed width applies with no query at all"},

		// The mirror rule for a fixed height
		{0, 200, "/f.png?width=100&height=50", 0, 200, "fixed height discards the requested width"},
		{0, 200, "/f.png", 0, 200, "fixed height applies with no query at all"},

		// With both pinned, nothing the caller sends matters
		{300, 200, "/f.png?width=1&height=1", 300, 200, "both fixed"},
		{300, 200, "/f.png", 300, 200, "both fixed, no query"},

		// Negative rules are not treated as rules, so the caller's values survive
		{-300, 0, "/f.png?width=100&height=50", 100, 50, "a negative rule is ignored"},

		// Unparseable query values become zero rather than an error
		{0, 0, "/f.png?width=abc&height=", 0, 0, "garbage parses as zero"},
		{0, 0, "/f.png?width=-100", -100, 0, "a negative request is passed through"},
	}

	for _, row := range table {

		rules := NewAttachmentRules()
		rules.Width = row.ruleWidth
		rules.Height = row.ruleHeight

		_, width, height, _ := fileSpecFor(t, rules, row.requested, ".png")

		require.Equal(t, row.width, width, "rule=%dx%d requested=%q (%s)", row.ruleWidth, row.ruleHeight, row.requested, row.reason)
		require.Equal(t, row.height, height, "rule=%dx%d requested=%q (%s)", row.ruleWidth, row.ruleHeight, row.requested, row.reason)
	}
}

// TestAttachmentRules_FileSpec_Bitrate confirms how a fixed bitrate overrides the caller
func TestAttachmentRules_FileSpec_Bitrate(t *testing.T) {

	rules := NewAttachmentRules()

	_, _, _, bitrate := fileSpecFor(t, rules, "/f.mp3?bitrate=64", ".mp3")
	require.Equal(t, 64, bitrate, "with no rule, the caller decides")

	rules.Bitrate = 128
	_, _, _, bitrate = fileSpecFor(t, rules, "/f.mp3?bitrate=64", ".mp3")
	require.Equal(t, 128, bitrate, "a fixed bitrate overrides the caller")

	_, _, _, bitrate = fileSpecFor(t, rules, "/f.mp3", ".mp3")
	require.Equal(t, 128, bitrate, "...and applies with no query at all")

	// Unlike width and height, the bitrate rule does not suppress anything else
	_, width, _, _ := fileSpecFor(t, rules, "/f.mp3?width=100", ".mp3")
	require.Equal(t, 100, width)
}

// TestAttachmentRules_FileSpec_Filename confirms how the request path becomes a filename
func TestAttachmentRules_FileSpec_Filename(t *testing.T) {

	rules := NewAttachmentRules()

	table := []struct {
		path     string
		expected string
		reason   string
	}{
		{"/@123/attachments/456/hero.png", "hero", "only the last path segment is used"},
		{"/hero.png", "hero", "a one-segment path"},
		{"/archive.tar.gz", "archive.tar", "only the final extension is split off"},
		{"/hero", "hero", "no extension to remove"},
		{"/", "", "an empty path yields an empty filename"},
		{"", "", "no path at all"},
		{"/dir/", "", "a trailing slash means no filename"},
		{"/.hidden", "", "a dotfile is all extension, so nothing is left"},
	}

	for _, row := range table {

		address, err := url.Parse(row.path)
		require.Nil(t, err)

		filespec := rules.FileSpec(address, ".png")
		require.Equal(t, row.expected, filespec.Filename, "path=%q (%s)", row.path, row.reason)
	}
}

// TestAttachmentRules_FileSpec_Invariants confirms the fields every FileSpec carries
func TestAttachmentRules_FileSpec_Invariants(t *testing.T) {

	rules := NewAttachmentRules()

	address, err := url.Parse("/hero.png")
	require.Nil(t, err)

	filespec := rules.FileSpec(address, ".jpeg")

	require.Equal(t, ".jpeg", filespec.OriginalExtension, "the original extension is echoed back verbatim")
	require.NotNil(t, filespec.Metadata, "Metadata must be usable without a nil check")
	require.Equal(t, 0, len(filespec.Metadata))
	require.True(t, filespec.Cache, "every rendition is cacheable")
	require.Equal(t, "hero.png", filespec.DownloadFilename())
}

// TestAttachmentRules_FileSpec_ReceiverIsNotMutated confirms that computing a FileSpec
// leaves the caller's rules untouched, even when the default list is filled in
func TestAttachmentRules_FileSpec_ReceiverIsNotMutated(t *testing.T) {

	rules := NewAttachmentRules()

	address, err := url.Parse("/hero.png")
	require.Nil(t, err)

	filespec := rules.FileSpec(address, ".png")
	require.Equal(t, ".png", filespec.Extension)

	// RULE: FileSpec has a value receiver, so the image defaults it computes are
	// scratch work.  If this ever fails, one request's defaults have leaked into
	// the Template's rules, and every later request inherits them.
	require.Equal(t, 0, len(rules.Extensions))
}

/******************************************
 * Fuzz Targets
 ******************************************/

// FuzzAttachmentRules_FileSpec confirms that no request URL can escape the extension
// allow-list or crash the server.  The path and query are attacker-controlled.
func FuzzAttachmentRules_FileSpec(f *testing.F) {

	f.Add("/hero.png", ".png", "webp,png")
	f.Add("/hero.jpg?width=100&height=50", ".jpeg", "")
	f.Add("/a/b/c/../../hero.gif?bitrate=999999999999", ".gif", "webp")
	f.Add("/", "", "")
	f.Add("/.hidden", ".png", "")
	f.Add("/hero.html", ".png", "webp")
	f.Add("/hero.png%00.html", ".png", "webp")
	f.Add("/hero.png?width=-1&height=-1", ".png", "")
	f.Add("/\xff\xfe.png", ".png", "")

	f.Fuzz(func(t *testing.T, path string, originalExtension string, extensionList string) {

		address, err := url.Parse(path)

		// Not every fuzzed string is a URL; those are the caller's problem, not ours
		if err != nil {
			return
		}

		// The allow-list comes from a Template, which is trusted configuration, so only
		// plausible extensions are fed in here.  The attacker-controlled half of this
		// function is the request URL.
		rules := NewAttachmentRules()
		for _, extension := range strings.Split(extensionList, ",") {
			if isPlausibleExtension(extension) {
				rules.Extensions = append(rules.Extensions, extension)
			}
		}

		filespec := rules.FileSpec(address, originalExtension)

		// PROPERTY: The extension is always dot-prefixed, and holds exactly one dot,
		// so it cannot be spliced into a second extension.
		require.True(t, strings.HasPrefix(filespec.Extension, "."), "extension=%q", filespec.Extension)
		require.Equal(t, 1, strings.Count(filespec.Extension, "."), "extension=%q", filespec.Extension)

		// PROPERTY: When a Template declares an allow-list, the result is on it.
		// This is the guarantee that keeps ".html" from being served as itself.
		if len(rules.Extensions) > 0 {
			require.True(t, slices.Contains(rules.Extensions, strings.TrimPrefix(filespec.Extension, ".")),
				"extension %q escaped the allow-list %v", filespec.Extension, rules.Extensions)
		}

		// PROPERTY: The FileSpec is always usable -- Metadata is never nil, and the
		// original extension is echoed back unchanged.
		require.NotNil(t, filespec.Metadata)
		require.Equal(t, originalExtension, filespec.OriginalExtension)

		// PROPERTY: The same request always produces the same FileSpec
		require.Equal(t, filespec, rules.FileSpec(address, originalExtension))
	})
}

// isPlausibleExtension returns TRUE if a string could be a real file extension
func isPlausibleExtension(value string) bool {

	if value == "" {
		return false
	}

	for _, character := range value {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789", character) {
			return false
		}
	}

	return true
}
