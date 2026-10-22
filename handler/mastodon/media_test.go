package mastodon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testPNG returns a real, valid PNG of the given size, so sniffMedia has genuine image bytes
// to detect -- not a hand-rolled header that happens to pass a magic-number check.
func testPNG(t *testing.T, width int, height int) []byte {

	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.White)

	var buffer bytes.Buffer
	require.NoError(t, png.Encode(&buffer, img))

	return buffer.Bytes()
}

func TestSniffMedia_DetectsRealImageDimensions(t *testing.T) {

	data := testPNG(t, 40, 30)

	reader, contentType, width, height, err := sniffMedia(bytes.NewReader(data))

	require.NoError(t, err)
	require.Equal(t, "image/png", contentType)
	require.Equal(t, 40, width)
	require.Equal(t, 30, height)

	// The full file must still be readable afterward -- sniffing peeks, it must not consume.
	replayed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, data, replayed)
}

func TestSniffMedia_NonImageHasNoDimensionsButStillReplays(t *testing.T) {

	data := []byte("just some plain text, not an image at all")

	reader, contentType, width, height, err := sniffMedia(bytes.NewReader(data))

	require.NoError(t, err)
	require.Contains(t, contentType, "text/plain")
	require.Equal(t, 0, width)
	require.Equal(t, 0, height)

	replayed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, data, replayed)
}

func TestSniffMedia_FileShorterThanThePeekWindow(t *testing.T) {

	data := []byte("tiny")

	reader, _, width, height, err := sniffMedia(bytes.NewReader(data))

	require.NoError(t, err)
	require.Equal(t, 0, width)
	require.Equal(t, 0, height)

	replayed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, data, replayed)
}

func TestMediaAttachmentType(t *testing.T) {

	require.Equal(t, "image", mediaAttachmentType("image/png"))
	require.Equal(t, "video", mediaAttachmentType("video/mp4"))
	require.Equal(t, "audio", mediaAttachmentType("audio/mpeg"))

	// Mastodon defines no "file"/"unknown" status-media type, so an unrecognized upload
	// (a PDF, an octet-stream) falls back to "image" rather than an empty/invalid type.
	require.Equal(t, "image", mediaAttachmentType("application/pdf"))
	require.Equal(t, "image", mediaAttachmentType(""))
}

func TestAttachmentToMediaAttachment(t *testing.T) {

	sized := model.NewAttachment(model.AttachmentObjectTypeUser, primitive.NewObjectID())
	sized.ContentType = "image/jpeg"
	sized.URL = "https://example.com/photo.jpg"
	sized.Description = "A photo"
	sized.Width = 800
	sized.Height = 600

	result := attachmentToMediaAttachment(sized)

	require.Equal(t, "image", result.Type)
	require.Equal(t, "https://example.com/photo.jpg", result.URL)
	require.Equal(t, "https://example.com/photo.jpg", result.PreviewURL) // images preview as themselves
	require.Equal(t, "A photo", result.Description)
	require.Equal(t, map[string]any{"width": 800, "height": 600}, result.Meta["original"])

	unsized := model.NewAttachment(model.AttachmentObjectTypeUser, sized.ObjectID)
	unsized.ContentType = "video/mp4"
	unsized.URL = "https://example.com/clip.mp4"

	videoResult := attachmentToMediaAttachment(unsized)

	require.Equal(t, "video", videoResult.Type)
	require.Empty(t, videoResult.PreviewURL) // no thumbnail to offer for video
	require.NotContains(t, videoResult.Meta, "original")
}
