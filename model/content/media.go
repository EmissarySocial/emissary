package content

import (
	"bytes"
	"html/template"
	"mime"
	"path"
	"strings"
)

// ImageTemplate represents the HTML that is returned for an image media type
var ImageTemplate *template.Template

// VideoTemplate represents the HTML that is returned for a video media type
var VideoTemplate *template.Template

func init() {
	ImageTemplate = template.Must(template.New("ImageTemplate").Parse(`<img src="{{.URL}}"{{if gt .Height 0}} height="{{.Height}}"{{end}}{{if gt .Width 0}} width="{{.Width}}"{{end}}>`))
	VideoTemplate = template.Must(template.New("VideoTemplate").Parse(`<video src="{{.URL}}"{{if gt .Height 0}} height="{{.Height}}"{{end}}{{if gt .Width 0}} width="{{.Width}}"{{end}}></video>`))
}

// Media represents any image or video media that is to be included in an HTML page.
type Media struct {
	URL    string // Publicly accessible URL of the media
	Type   string // Saved MimeType of the media
	Height int    // Height (in pixels) of the media
	Width  int    // Width (in pixels) of the media
}

// HTML implements the HTMLer interfacd
func (media Media) HTML() string {

	buffer := bytes.Buffer{}

	var template *template.Template

	switch media.MimeCategory() {
	case "image":
		template = ImageTemplate
	case "video":
		template = VideoTemplate
	default:
		return ""
	}

	if err := template.Execute(&buffer, media); err != nil {
		return ""
	}

	return buffer.String()
}

// WebComponents accumulates all of the scripts that are required to correctly render the HTML for this content object
func (media *Media) WebComponents(accumulator map[string]bool) {

	switch media.MimeCategory() {
	case "image":
		accumulator["/components/content-image.js"] = true
	case "video":
		accumulator["/components/content-video.js"] = true

	}
	return
}

// Extension safely identifies the file extension from a URL
func (media Media) Extension() string {

	result := path.Ext(media.URL)

	return result
}

// MimeType calculates the MIME type of this value
func (media Media) MimeType() string {

	// If we have a type in the object already, then use that.
	if media.Type != "" {
		return media.Type
	}

	// Otherwise, use the built-in system function to determine it using the file URL.
	return mime.TypeByExtension(media.Extension())
}

// MimeCategory returns the first part of the Mime Type (before the /).  This gives us a higher level category to use when representing MimeTypes as HTML
func (media Media) MimeCategory() string {

	// Get the full mime type for this media
	mimeType := media.MimeType()

	// Try to return only the FIRST part of the mime type
	if index := strings.Index(mimeType, "/"); index >= 0 {
		return mimeType[:index]
	}

	// Fall through to here means that we don't have a valid mime type.  So, return empty string instead.
	return ""
}
