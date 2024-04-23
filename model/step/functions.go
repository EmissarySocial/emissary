package step

import (
	"html/template"

	"github.com/EmissarySocial/emissary/tools/templates"
)

func FuncMap() template.FuncMap {
	return templates.FuncMap(nil)
}
