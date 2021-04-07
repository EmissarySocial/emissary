package vocabulary

import "github.com/benpate/form"

func All(library form.Library) {
	LayoutGroup(library)
	LayoutHorizontal(library)
	LayoutVertical(library)
	Option(library)
	Select(library)
	Text(library)
	Textarea(library)
}
