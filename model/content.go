package model

const ContentFormatHTML = "HTML"
const ContentFormatText = "TEXT"
const ContentFormatMedia = "MEDIA"
const ContentFormatTabs = "TABS"
const ContentFormatContainer = "CONTAINER"
const ContentFormatEncrypted = "ENCRYPTED"

// Content represents a piece of page content that can be stored in the system.
type Content struct {
	Type   string // Type identifies what kind of template to use when rendering this content
	Format string // Format identifies the format of the data contained in this content
	Data   map[string]interface{}
}

// Get returns the data property of this content.  If the property is not present, then it returns Nil.
func (content Content) Get(value string) interface{} {

	if result, ok := content.Data[value]; ok {
		return result
	}

	return nil
}
