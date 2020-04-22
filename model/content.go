package model

const ContentTypeHTML = "HTML"
const ContentTypeText = "TEXT"
const ContentTypeMedia = "MEDIA"
const ContentTypeTabs = "TABS"
const ContentTypeContainer = "CONTAINER"

// Content represents a piece of page content that can be stored in the system.
type Content struct {
	Type           string
	RenderTemplate string
}
