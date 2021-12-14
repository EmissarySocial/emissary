package service

import (
	"html/template"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// Layout service manages the global site layout that is stored in a particular path of the
// filesystem.
type Layout struct {
	path    string
	funcMap template.FuncMap
	domain  model.Layout
	global  model.Layout
	group   model.Layout
	user    model.Layout
}

// NewLayout returns a fully initialized Layout service.
func NewLayout(path string, funcMap template.FuncMap) *Layout {

	service := &Layout{
		path:    path,
		funcMap: funcMap,
	}

	return service
}

func (service *Layout) Global() model.Layout {
	return service.global
}

func (service *Layout) Domain() model.Layout {
	return service.domain
}

func (service *Layout) User() model.Layout {
	return service.user
}

// getTemplateFromFilesystem retrieves the template from the disk and parses it into
func (service *Layout) getTemplateFromFilesystem(folder string) error {

	layoutID := strings.TrimPrefix(folder, "_")
	path := service.path + "/" + folder
	layout := model.NewLayout(layoutID)

	if err := loadTemplateFromFilesystem(path, service.funcMap, layout.HTMLTemplate); err != nil {
		return derp.Wrap(err, "ghost.service.layout.getTemplateFromFilesystem", "Error loading Schema", layoutID)
	}

	if err := loadSchemaFromFilesystem(path, &layout.Schema); err != nil {
		return derp.Wrap(err, "ghost.service.layout.getTemplateFromFilesystem", "Error loading Schema", layoutID)
	}

	switch layoutID {
	case "global":
		service.global = layout
	case "group":
		service.group = layout
	case "user":
		service.user = layout
	}

	return nil
}
