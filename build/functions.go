package build

import (
	"html/template"

	"github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/icon"
)

func FuncMap(icons icon.Provider) template.FuncMap {
	return templates.FuncMap(icons)
}
