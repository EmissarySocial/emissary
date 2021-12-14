package service

import (
	"html/template"
	"io/ioutil"

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

	// Load all templates from the filesystem
	list, err := ioutil.ReadDir(path)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.templateSource.File.List", "Error listing files in Filesystem", path))
		panic("Error listing files in Filesystem")
	}

	// Use a separate counter because not all files will be included in the result
	for _, fileInfo := range list {

		if !fileInfo.IsDir() {
			continue
		}

		folderName := fileInfo.Name()

		// Add all other directories into the Template service as Templates
		if err := service.loadFromFilesystem(folderName); err != nil {
			derp.Report(derp.Wrap(err, "ghost.service.layout.NewLayout", "Error loading Layout from Filesystem"))
			panic("Error loading Layout from Filesystem")
		}
	}

	return service
}

func (service *Layout) Global() model.Layout {
	return service.global
}

func (service *Layout) Group() model.Layout {
	return service.group
}

func (service *Layout) Domain() model.Layout {
	return service.domain
}

func (service *Layout) User() model.Layout {
	return service.user
}

// getTemplateFromFilesystem retrieves the template from the disk and parses it into
func (service *Layout) loadFromFilesystem(folderName string) error {

	path := service.path + "/" + folderName
	layout := model.NewLayout(folderName, service.funcMap)

	// System folders (except for "static" and "global") have a schema.json file
	if (folderName != "static") && (folderName != "global") {
		if err := loadModelFromFilesystem(path, &layout); err != nil {
			return derp.Wrap(err, "ghost.service.layout.getTemplateFromFilesystem", "Error loading Schema", folderName)
		}
	}

	if err := loadHTMLTemplateFromFilesystem(path, layout.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, "ghost.service.layout.getTemplateFromFilesystem", "Error loading Schema", folderName)
	}

	switch folderName {

	case "global":
		service.global = layout
	case "group":
		service.group = layout
	case "user":
		service.user = layout
	}

	return nil
}
