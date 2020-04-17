package content

// GalleryFormatGrid represents a thumbnail grid layout for content galleries.
const GalleryFormatGrid = "GRID"

// GalleryFormatSlider represents a single large image in a gallery that can be swiped or clicked to scroll to the next.
const GalleryFormatSlider = "SLIDER"

// Gallery displays one or more media items in a nice-looking format
type Gallery struct {
	Format string
	Media  []Media
}

// HTML implements the HTMLer interface
func (gallery *Gallery) HTML() string {
	return "MEDIA GALLERY HERE"
}

// WebComponents accumulates all of the scripts that are required to correctly render the HTML for this content object
func (gallery *Gallery) WebComponents(accumulator map[string]bool) {
	accumulator["/components/gallery.js"] = true

	for _, content := range gallery.Media {
		content.WebComponents(accumulator)
	}
}
