package vocabulary

import "github.com/benpate/form"

// All registers all available widgets with the form library
func All(library form.Library) {
	LayoutGroup(library)
	LayoutHorizontal(library)
	LayoutVertical(library)
	Text(library)
	Textarea(library)
}
