package step

import (
	"html/template"

	"github.com/EmissarySocial/emissary/tools/templates"
)

// FuncMap returns the template helper functions available to step configuration templates
func FuncMap() template.FuncMap {
	return templates.FuncMap(nil)
}
