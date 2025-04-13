package model

import (
	"mime"
	"net/url"
	"slices"

	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// AttachmentRules defines the rules for downloading an attachment
type AttachmentRules struct {
	Extensions sliceof.String // Allowed extensions.  The first value is used as the default.
	Width      int            // Fixed width for all downloads
	Height     int            // Fixed height for all downloads
	Bitrate    int
}

// NewAttachmentRules returns a fully initialized AttachmentRules object
func NewAttachmentRules() AttachmentRules {
	return AttachmentRules{
		Extensions: []string{},
		Width:      0,
		Height:     0,
		Bitrate:    0,
	}
}

// FileSpec applies the attachment rules to a request, and returns the best-matching FileSpec definition for mediaserver
func (rules AttachmentRules) FileSpec(address *url.URL, originalExtension string) mediaserver.FileSpec {

	// Get path values
	path := address.Path
	fullname := list.Slash(path).Last()
	filename, extension := list.Dot(fullname).SplitTail()

	// Get query values
	query := address.Query()
	height := convert.Int(query.Get("height"))
	width := convert.Int(query.Get("width"))
	bitrate := convert.Int(query.Get("bitrate"))

	// If Width is defined, use that.
	if rules.Width > 0 {
		width = rules.Width

		// If no height is defined, then don't allow height parameters
		if rules.Height == 0 {
			height = 0
		}
	}

	// If height is defined, use that.
	if rules.Height > 0 {
		height = rules.Height

		// If no width is defined, then don't allow width parameters
		if rules.Width == 0 {
			width = 0
		}
	}

	if rules.Bitrate > 0 {
		bitrate = rules.Bitrate
	}

	originalType := mime.TypeByExtension(originalExtension)
	originalCategory := list.Slash(originalType).First()

	// Map duplicate extensions to the canonical value
	switch extension {
	case "jpg":
		extension = "jpeg"
	}

	// Calculate default types if none is provided
	if len(rules.Extensions) == 0 {

		switch originalCategory {

		case "image":
			rules.Extensions = []string{"webp", "png", "jpeg", "gif"}

		case "video":
			rules.Extensions = []string{"mp4", "webm", "ogv"}

		case "audio":
			rules.Extensions = []string{"mp3", "opus", "aac", "ogg", "flac"}
		}
	}

	// Guarantee that the requested extension is allowed.
	// If not, force the default extension
	if len(rules.Extensions) > 0 {
		if !slices.Contains(rules.Extensions, extension) {
			extension = rules.Extensions[0]
		}
	}

	extension = "." + extension

	// Return the "cleaned" mediaserver.FileSpec object
	result := mediaserver.FileSpec{
		Filename:          filename.String(),
		Extension:         extension,
		Width:             width,
		Height:            height,
		Bitrate:           bitrate,
		OriginalExtension: originalExtension,
		Metadata:          mapof.NewString(),
		Cache:             true,
	}

	return result
}
