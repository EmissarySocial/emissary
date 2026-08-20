package model

import (
	"net/url"
	"strings"
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Constructors
 ******************************************/

// TestNewAttachment confirms that a new Attachment is fully initialized and ready to save
func TestNewAttachment(t *testing.T) {

	objectID := primitive.NewObjectID()
	attachment := NewAttachment(AttachmentObjectTypeStream, objectID)

	require.False(t, attachment.AttachmentID.IsZero(), "constructor must mint an ID")
	require.Equal(t, AttachmentObjectTypeStream, attachment.ObjectType)
	require.Equal(t, objectID, attachment.ObjectID)

	// Rules must be initialized, not nil, so that schema writes into them do not panic
	require.NotNil(t, attachment.Rules.Extensions)
	require.Equal(t, 0, len(attachment.Rules.Extensions))

	// Two Attachments must never share an ID
	other := NewAttachment(AttachmentObjectTypeStream, objectID)
	require.NotEqual(t, attachment.AttachmentID, other.AttachmentID)
}

// TestNewEmptyAttachment confirms that an empty Attachment carries no ID and no rules
func TestNewEmptyAttachment(t *testing.T) {

	attachment := NewEmptyAttachment()

	require.True(t, attachment.AttachmentID.IsZero())
	require.True(t, attachment.ObjectID.IsZero())
	require.Equal(t, "", attachment.ObjectType)
	require.Equal(t, "", attachment.Original)
	require.Equal(t, "", attachment.ContentType)
	require.Equal(t, "", attachment.URL)
	require.Equal(t, "", attachment.Status)
	require.Equal(t, 0, attachment.Width)
	require.Equal(t, 0, attachment.Height)
	require.Equal(t, 0, attachment.Duration)
	require.Equal(t, 0, attachment.Rank)

	// Unlike NewAttachment, the rules are left nil
	require.Nil(t, attachment.Rules.Extensions)
}

/******************************************
 * data.Object Interface
 ******************************************/

// TestAttachment_ID confirms that ID() is the hex form of the AttachmentID
func TestAttachment_ID(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())
	require.Equal(t, attachment.AttachmentID.Hex(), attachment.ID())

	// An unsaved Attachment still returns a (zero) hex string, never an empty one
	empty := NewEmptyAttachment()
	require.Equal(t, "000000000000000000000000", empty.ID())
}

/******************************************
 * AccessLister Interface
 ******************************************/

// TestAttachment_State confirms that Attachments have exactly one state
func TestAttachment_State(t *testing.T) {
	require.Equal(t, "default", NewEmptyAttachment().State())
}

// TestAttachment_IsAuthor confirms that Attachments never report an author, for any UserID
func TestAttachment_IsAuthor(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeUser, primitive.NewObjectID())

	require.False(t, attachment.IsAuthor(primitive.NewObjectID()))
	require.False(t, attachment.IsAuthor(primitive.NilObjectID))

	// Not even the owner of the Attachment counts as its author
	require.False(t, attachment.IsAuthor(attachment.ObjectID))
}

// TestAttachment_IsMyself confirms that only a User's own Attachment represents that User
func TestAttachment_IsMyself(t *testing.T) {

	userID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()

	table := []struct {
		objectType string
		objectID   primitive.ObjectID
		userID     primitive.ObjectID
		expected   bool
		reason     string
	}{
		{AttachmentObjectTypeUser, userID, userID, true, "a User's own avatar"},
		{AttachmentObjectTypeUser, userID, otherID, false, "another User's avatar"},
		{AttachmentObjectTypeStream, userID, userID, false, "matching IDs, but a Stream attachment"},
		{AttachmentObjectTypeDomain, userID, userID, false, "matching IDs, but a Domain attachment"},
		{AttachmentObjectTypeSearchTag, userID, userID, false, "matching IDs, but a SearchTag attachment"},
		{"", userID, userID, false, "matching IDs, but no object type"},

		// RULE: A zero UserID is an anonymous visitor, who is nobody.  Without this
		// guard, every unowned Attachment would claim to be the anonymous visitor.
		{AttachmentObjectTypeUser, primitive.NilObjectID, primitive.NilObjectID, false, "anonymous visitor, unowned attachment"},
		{AttachmentObjectTypeUser, userID, primitive.NilObjectID, false, "anonymous visitor, owned attachment"},
	}

	for _, row := range table {

		attachment := NewAttachment(row.objectType, row.objectID)

		require.Equal(t, row.expected, attachment.IsMyself(row.userID),
			"objectType=%q (%s)", row.objectType, row.reason)
	}
}

// TestAttachment_RolesToGroupIDs confirms that Attachments grant only the two magic groups
func TestAttachment_RolesToGroupIDs(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeUser, primitive.NewObjectID())

	require.Equal(t, Permissions{}, attachment.RolesToGroupIDs())
	require.Equal(t, Permissions{MagicGroupIDAnonymous}, attachment.RolesToGroupIDs(MagicRoleAnonymous))
	require.Equal(t, Permissions{MagicGroupIDAuthenticated}, attachment.RolesToGroupIDs(MagicRoleAuthenticated))

	// RULE: Attachments pass a NilObjectID as the owner, so "self" and "author"
	// resolve to nobody, even on an Attachment that a User owns.
	require.Equal(t, Permissions{}, attachment.RolesToGroupIDs(MagicRoleMyself))
	require.Equal(t, Permissions{}, attachment.RolesToGroupIDs(MagicRoleAuthor))

	// Unrecognized roles are dropped, and the recognized ones keep their order
	require.Equal(t,
		Permissions{MagicGroupIDAnonymous, MagicGroupIDAuthenticated},
		attachment.RolesToGroupIDs(MagicRoleAnonymous, "not-a-real-role", MagicRoleAuthenticated))
}

// TestAttachment_RolesToPrivilegeIDs confirms that Attachments never grant a Circle or Product
func TestAttachment_RolesToPrivilegeIDs(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeUser, primitive.NewObjectID())

	require.Equal(t, Permissions{}, attachment.RolesToPrivilegeIDs())
	require.Equal(t, Permissions{}, attachment.RolesToPrivilegeIDs(MagicRoleAnonymous, MagicRoleAuthenticated, MagicRoleMyself))
}

/******************************************
 * URLs and File Names
 ******************************************/

// TestAttachment_CalcURL confirms the public URL that each kind of owner produces
func TestAttachment_CalcURL(t *testing.T) {

	objectID := primitive.NewObjectID()

	table := []struct {
		objectType string
		expected   string
	}{
		{AttachmentObjectTypeUser, "https://example.com/@" + objectID.Hex() + "/attachments/"},
		{AttachmentObjectTypeDomain, "https://example.com/.domain/attachments/"},
		{AttachmentObjectTypeStream, "https://example.com/" + objectID.Hex() + "/attachments/"},
		{AttachmentObjectTypeSearchTag, "https://example.com/" + objectID.Hex() + "/attachments/"},
		{"", "https://example.com/" + objectID.Hex() + "/attachments/"},
	}

	for _, row := range table {
		attachment := NewAttachment(row.objectType, objectID)
		require.Equal(t, row.expected+attachment.AttachmentID.Hex(), attachment.CalcURL("https://example.com"),
			"objectType=%q", row.objectType)
	}

	// The host is pasted on verbatim, so an empty host yields a relative URL
	attachment := NewAttachment(AttachmentObjectTypeDomain, objectID)
	require.Equal(t, "/.domain/attachments/"+attachment.AttachmentID.Hex(), attachment.CalcURL(""))

	// RULE: Domain attachments must not leak the owner's ObjectID into the URL
	require.NotContains(t, attachment.CalcURL("https://example.com"), objectID.Hex())
}

// TestAttachment_OriginalExtension confirms how a filename is split into an extension
func TestAttachment_OriginalExtension(t *testing.T) {

	table := []struct {
		original string
		expected string
		reason   string
	}{
		{"photo.png", ".png", "ordinary filename"},
		{"photo.PNG", ".PNG", "case is preserved here, not folded"},
		{"archive.tar.gz", ".gz", "only the last segment counts"},
		{"noextension", ".noextension", "with no dot, the whole name reads as the extension"},
		{".hidden", ".hidden", "a dotfile is all extension"},
		{"photo.", ".", "a trailing dot leaves nothing behind it"},
		{"", ".", "an empty name is a bare dot"},
		{"weird name.jpeg", ".jpeg", "spaces are irrelevant"},
	}

	for _, row := range table {
		attachment := NewEmptyAttachment()
		attachment.Original = row.original
		require.Equal(t, row.expected, attachment.OriginalExtension(), "original=%q (%s)", row.original, row.reason)
	}
}

// TestAttachment_DownloadExtension confirms which uploads are re-encoded to WebP
func TestAttachment_DownloadExtension(t *testing.T) {

	table := []struct {
		original string
		expected string
		reason   string
	}{
		{"photo.jpg", ".webp", "JPEG is re-encoded"},
		{"photo.jpeg", ".webp", "JPEG (long form) is re-encoded"},
		{"photo.png", ".webp", "PNG is re-encoded"},
		{"photo.JPG", ".webp", "the extension is lowercased before matching"},
		{"photo.PnG", ".webp", "...in any mix of case"},
		{"photo.gif", ".gif", "GIF is served as-is"},
		{"photo.webp", ".webp", "WebP is already WebP"},
		{"clip.MP4", ".mp4", "non-image extensions are only lowercased"},
		{"doc.pdf", ".pdf", "documents pass through"},
		{"noextension", ".noextension", "no dot, no change"},
		{"", ".", "empty name yields a bare dot"},
	}

	for _, row := range table {
		attachment := NewEmptyAttachment()
		attachment.Original = row.original
		require.Equal(t, row.expected, attachment.DownloadExtension(), "original=%q (%s)", row.original, row.reason)
	}
}

// TestAttachment_DownloadMimeType confirms the type an Attachment is served with
func TestAttachment_DownloadMimeType(t *testing.T) {

	// Only extensions from Go's built-in table are asserted here.  Types such as
	// .mp3 and .mp4 are read from the host's mime database, which varies by machine.
	table := []struct {
		original string
		expected string
	}{
		{"photo.jpg", "image/webp"},
		{"photo.png", "image/webp"},
		{"photo.gif", "image/gif"},
		{"doc.pdf", "application/pdf"},
		{"page.html", "text/html; charset=utf-8"},
		{"unknown.qqq", ""},
		{"", ""},
	}

	for _, row := range table {
		attachment := NewEmptyAttachment()
		attachment.Original = row.original
		require.Equal(t, row.expected, attachment.DownloadMimeType(), "original=%q", row.original)
	}

	// RULE: The served type follows the *download* extension, not the original,
	// so a JPEG upload must never be announced as image/jpeg.
	attachment := NewEmptyAttachment()
	attachment.Original = "photo.jpg"
	require.Equal(t, "image/jpeg", attachment.OriginalMimeType())
	require.Equal(t, "image/webp", attachment.DownloadMimeType())
}

/******************************************
 * Mime Types
 ******************************************/

// TestAttachment_MimeType confirms that the sniffed ContentType outranks the filename.
func TestAttachment_MimeType(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())
	attachment.Original = "poc.html"

	// With no sniffed type, the filename is all we have to go on.
	require.Equal(t, "text/html; charset=utf-8", attachment.MimeType())

	// Once the bytes have been sniffed, the filename stops being the answer.
	attachment.ContentType = "image/gif"
	require.Equal(t, "image/gif", attachment.MimeType())
	require.Equal(t, "image", attachment.MimeCategory())

	// ...but the claim the filename makes is still available to whoever needs it.
	require.Equal(t, "text/html; charset=utf-8", attachment.OriginalMimeType())
}

// TestAttachment_MimeCategory confirms the leading half of an Attachment's mime type
func TestAttachment_MimeCategory(t *testing.T) {

	table := []struct {
		original    string
		contentType string
		expected    string
		reason      string
	}{
		{"photo.png", "image/png", "image", "sniffed type wins"},
		{"photo.png", "", "image", "falls back to the filename"},
		{"doc.pdf", "", "application", "application category"},
		{"page.html", "", "text", "the charset parameter stays out of the category"},
		{"", "", "", "nothing to categorize"},
		{"", "image", "image", "a type with no slash is all category"},
		{"", "/png", "", "a leading slash means an empty category"},
	}

	for _, row := range table {
		attachment := NewEmptyAttachment()
		attachment.Original = row.original
		attachment.ContentType = row.contentType
		require.Equal(t, row.expected, attachment.MimeCategory(),
			"original=%q contentType=%q (%s)", row.original, row.contentType, row.reason)
	}
}

// TestAttachment_CanServeInline confirms which Attachments a browser is allowed to render.
func TestAttachment_CanServeInline(t *testing.T) {

	table := []struct {
		original    string
		contentType string
		expected    bool
		reason      string
	}{
		// Media whose filename and contents agree: MediaServer re-encodes these, so the
		// bytes the client receives are generated by FFmpeg and cannot carry script.
		{"photo.png", "image/png", true, "real image"},
		{"photo.JPG", "image/jpeg", true, "real image, shouty extension"},
		{"clip.mp4", "video/mp4", true, "real video"},
		{"song.mp3", "audio/mpeg", true, "real audio"},

		// The categories only have to agree with each other, not the exact types
		{"photo.png", "image/gif", true, "image name, different image contents"},

		// The reported stored-XSS payload.
		{"poc.html", "text/html", false, "HTML is copied verbatim, never rendered"},

		// The polyglot bypass: GIF magic bytes satisfy an "image/*" accept-type on upload,
		// but the .html name means MediaServer copies it verbatim instead of re-encoding.
		{"poc.html", "image/gif", false, "contents say image, filename says HTML"},

		// The mirror image: an HTML document wearing an image filename.
		{"poc.png", "text/html", false, "filename says image, contents say HTML"},

		// Crossing media categories is fine: MediaServer runs the file through FFmpeg
		// either way, so the bytes the browser receives are still FFmpeg's output.
		{"photo.png", "video/mp4", true, "image name, video contents"},
		{"clip.mp4", "image/png", true, "video name, image contents"},

		// Files MediaServer never re-encodes are downloads, whatever they contain.
		{"doc.pdf", "application/pdf", false, "PDF is not re-encoded"},
		{"archive.zip", "application/zip", false, "archive is not re-encoded"},
		{"noextension", "application/octet-stream", false, "no extension to type it by"},
		{"", "", false, "no filename at all"},
		// SVG can carry script, but it never reaches the browser as SVG: the rules
		// force the download extension to .webp, so FFmpeg either re-encodes it or
		// fails, and the raw markup is never served.
		{"poc.svg", "image/svg+xml", true, "SVG is re-encoded, not passed through"},

		// Attachments uploaded before content sniffing have no ContentType, so the
		// filename alone decides -- exactly the behavior these records shipped with.
		{"legacy.png", "", true, "legacy image"},
		{"legacy.html", "", false, "legacy HTML"},
	}

	for _, row := range table {

		attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())
		attachment.Original = row.original
		attachment.ContentType = row.contentType

		require.Equal(t, row.expected, attachment.CanServeInline(),
			"original=%q contentType=%q (%s)", row.original, row.contentType, row.reason)
	}
}

// TestIsInlineMediaCategory confirms the exact set of categories MediaServer re-encodes
func TestIsInlineMediaCategory(t *testing.T) {

	for _, category := range []string{"image", "audio", "video"} {
		require.True(t, isInlineMediaCategory(category), "category=%q", category)
	}

	for _, category := range []string{"", "application", "text", "font", "model", "multipart", "message",
		"Image", "IMAGE", "image/png", " image", "image "} {
		require.False(t, isInlineMediaCategory(category), "category=%q", category)
	}
}

/******************************************
 * Dimensions
 ******************************************/

// TestAttachment_HasDimensions confirms that both dimensions are required, not just one
func TestAttachment_HasDimensions(t *testing.T) {

	table := []struct {
		width    int
		height   int
		expected bool
	}{
		{640, 480, true},
		{1, 1, true},
		{0, 480, false},
		{640, 0, false},
		{0, 0, false},

		// Negative dimensions are nonsense, but they are non-zero, so they pass.
		// Pinned here so a future guard is a visible change, not a silent one.
		{-640, 480, true},
		{640, -480, true},
	}

	for _, row := range table {
		attachment := NewEmptyAttachment()
		attachment.Width = row.width
		attachment.Height = row.height
		require.Equal(t, row.expected, attachment.HasDimensions(), "%dx%d", row.width, row.height)
	}
}

// TestAttachment_AspectRatio confirms the CSS-ready ratio produced for each dimension pair
func TestAttachment_AspectRatio(t *testing.T) {

	table := []struct {
		width    int
		height   int
		expected string
		reason   string
	}{
		{1920, 1080, "1.7777777777777777", "16:9 widescreen"},
		{640, 480, "1.3333333333333333", "4:3"},
		{500, 500, "1", "square"},
		{1, 3, "0.3333333333333333", "tall images must not round to zero"},
		{1, 2, "0.5", "portrait"},
		{3, 1, "3", "landscape"},

		// RULE: A missing dimension yields "auto", which is valid CSS.  Several
		// templates interpolate this without a HasDimensions guard, so an empty
		// string here would emit "aspect-ratio:;" instead.
		{0, 1080, "auto", "no width"},
		{1920, 0, "auto", "no height -- this used to divide by zero"},
		{0, 0, "auto", "no dimensions at all"},

		// Non-zero nonsense still computes rather than panicking
		{-1920, 1080, "-1.7777777777777777", "negative width"},
		{1920, -1080, "-1.7777777777777777", "negative height"},
	}

	for _, row := range table {
		attachment := NewEmptyAttachment()
		attachment.Width = row.width
		attachment.Height = row.height
		require.Equal(t, row.expected, attachment.AspectRatio(), "%dx%d (%s)", row.width, row.height, row.reason)
	}
}

/******************************************
 * FileSpec and Rules
 ******************************************/

// TestAttachment_FileSpec confirms how an Attachment turns a request URL into a FileSpec
func TestAttachment_FileSpec(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())
	attachment.Original = "photo.png"

	// A nil address stands in for "just give me the default rendition"
	{
		filespec := attachment.FileSpec(nil)

		require.Equal(t, attachment.AttachmentID.Hex(), filespec.Filename, "the substitute path is the AttachmentID")
		require.Equal(t, ".webp", filespec.Extension, "images default to the first allowed extension")
		require.Equal(t, ".png", filespec.OriginalExtension)
		require.Equal(t, 0, filespec.Width)
		require.Equal(t, 0, filespec.Height)
		require.True(t, filespec.Cache)
		require.NotNil(t, filespec.Metadata)
	}

	// A real address contributes its filename and its query parameters
	{
		address, err := url.Parse("/@123/attachments/456/hero.png?width=100&height=50")
		require.Nil(t, err)

		filespec := attachment.FileSpec(address)

		require.Equal(t, "hero", filespec.Filename)
		require.Equal(t, ".png", filespec.Extension)
		require.Equal(t, 100, filespec.Width)
		require.Equal(t, 50, filespec.Height)
	}

	// The Attachment's own rules override anything the caller asks for
	{
		attachment.SetRules(300, 200, []string{"webp"})

		address, err := url.Parse("/hero.png?width=100&height=50")
		require.Nil(t, err)

		filespec := attachment.FileSpec(address)

		require.Equal(t, 300, filespec.Width)
		require.Equal(t, 200, filespec.Height)
		require.Equal(t, ".webp", filespec.Extension, "png is not in the allow-list")
	}
}

// TestAttachment_SetRules confirms that SetRules replaces every rule it owns
func TestAttachment_SetRules(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())
	attachment.Rules.Bitrate = 128

	attachment.SetRules(640, 480, []string{"webp", "png"})

	require.Equal(t, 640, attachment.Rules.Width)
	require.Equal(t, 480, attachment.Rules.Height)
	require.Equal(t, []string{"webp", "png"}, []string(attachment.Rules.Extensions))

	// SetRules does not touch the bitrate
	require.Equal(t, 128, attachment.Rules.Bitrate)

	// Calling it again replaces, rather than merges
	attachment.SetRules(0, 0, nil)
	require.Equal(t, 0, attachment.Rules.Width)
	require.Equal(t, 0, attachment.Rules.Height)
	require.Nil(t, attachment.Rules.Extensions)
}

/******************************************
 * JSON-LD
 ******************************************/

// TestAttachment_JSONLD confirms the ActivityStreams document produced for an Attachment
func TestAttachment_JSONLD(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())
	attachment.Original = "photo.jpg"
	attachment.URL = "https://example.com/photo"
	attachment.Description = "A very good photograph"
	attachment.Label = "Photo"
	attachment.Category = "Gallery"

	result := attachment.JSONLD()

	require.Equal(t, vocab.ObjectTypeDocument, result[vocab.PropertyType])
	require.Equal(t, "image/webp", result[vocab.PropertyMediaType], "federated peers are told the served type")
	require.Equal(t, "https://example.com/photo", result[vocab.PropertyURL])
	require.Equal(t, "A very good photograph", result[vocab.PropertyName], "description outranks label and category")

	// RULE: Dimensions are published only when both are known
	require.NotContains(t, result, "width")
	require.NotContains(t, result, "height")

	attachment.Width = 1920
	attachment.Height = 1080
	result = attachment.JSONLD()
	require.Equal(t, 1920, result["width"])
	require.Equal(t, 1080, result["height"])

	// A half-measured Attachment publishes neither dimension
	attachment.Height = 0
	result = attachment.JSONLD()
	require.NotContains(t, result, "width")
	require.NotContains(t, result, "height")
}

// TestAttachment_JSONLD_Name confirms the fallback order used to name an Attachment
func TestAttachment_JSONLD_Name(t *testing.T) {

	table := []struct {
		description string
		label       string
		category    string
		expected    string
	}{
		{"DESCRIPTION", "LABEL", "CATEGORY", "DESCRIPTION"},
		{"", "LABEL", "CATEGORY", "LABEL"},
		{"", "", "CATEGORY", "CATEGORY"},
		{"", "", "", ""},
	}

	for _, row := range table {
		attachment := NewEmptyAttachment()
		attachment.Description = row.description
		attachment.Label = row.label
		attachment.Category = row.category

		require.Equal(t, row.expected, attachment.JSONLD()[vocab.PropertyName],
			"description=%q label=%q category=%q", row.description, row.label, row.category)
	}
}

// TestAttachment_JSONLD_Empty confirms that an empty Attachment still produces a valid document
func TestAttachment_JSONLD_Empty(t *testing.T) {

	result := NewEmptyAttachment().JSONLD()

	require.Equal(t, vocab.ObjectTypeDocument, result[vocab.PropertyType])
	require.Equal(t, "", result[vocab.PropertyMediaType])
	require.Equal(t, "", result[vocab.PropertyURL])
	require.Equal(t, "", result[vocab.PropertyName])
	require.Equal(t, 4, len(result), "an empty Attachment publishes no dimensions")
}

/******************************************
 * Fuzz Targets
 ******************************************/

// FuzzAttachment_CanServeInline hunts for any filename/content-type pair that lets a
// non-media file be rendered by the browser.  Both inputs are attacker-controlled.
func FuzzAttachment_CanServeInline(f *testing.F) {

	f.Add("photo.png", "image/png")
	f.Add("poc.html", "image/gif")
	f.Add("poc.png", "text/html")
	f.Add("poc.svg", "image/svg+xml")
	f.Add("legacy.png", "")
	f.Add("", "")
	f.Add("photo.png\x00.html", "image/png")
	f.Add("photo.png;.html", "image/png")
	f.Add("\xff\xfe.png", "image/png")

	f.Fuzz(func(t *testing.T, original string, contentType string) {

		attachment := NewEmptyAttachment()
		attachment.Original = original
		attachment.ContentType = contentType

		if !attachment.CanServeInline() {
			return
		}

		// PROPERTY: Anything served inline must be named as a category that MediaServer
		// re-encodes.  If this ever fails, a raw upload is being handed to the browser.
		require.True(t, isInlineMediaCategory(attachment.MimeCategory()),
			"served inline as %q", attachment.MimeCategory())

		// PROPERTY: The name and the contents must agree.  A file that claims one
		// category and contains another is the polyglot bypass.
		if contentType != "" {
			nameCategory := strings.SplitN(attachment.OriginalMimeType(), "/", 2)[0]
			require.True(t, isInlineMediaCategory(nameCategory), "filename category %q", nameCategory)
		}
	})
}

// FuzzAttachment_AspectRatio confirms that no pair of dimensions can panic or emit invalid CSS
func FuzzAttachment_AspectRatio(f *testing.F) {

	f.Add(1920, 1080)
	f.Add(0, 0)
	f.Add(1920, 0)
	f.Add(0, 1080)
	f.Add(-1, -1)
	f.Add(1, 1<<62)

	f.Fuzz(func(t *testing.T, width int, height int) {

		attachment := NewEmptyAttachment()
		attachment.Width = width
		attachment.Height = height

		ratio := attachment.AspectRatio()

		// PROPERTY: The result is always a usable CSS value, never empty
		require.NotEmpty(t, ratio)

		// PROPERTY: "auto" is returned exactly when a dimension is missing
		require.Equal(t, width == 0 || height == 0, ratio == "auto", "%dx%d -> %q", width, height, ratio)

		// PROPERTY: Anything else parses back as a real number
		if ratio != "auto" {
			require.NotContains(t, ratio, "NaN")
			require.NotContains(t, ratio, "Inf")
		}
	})
}
