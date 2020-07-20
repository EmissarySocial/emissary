package templateSource

import (
	"encoding/json"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// File is a TemplateSource adapter that can load/save Templates from/to the local filesytem.
type File struct {
	Path            string
	TemplateService TemplateService
}

// NewFile returns a fully initialized File adapter for loading/saving Templates
func NewFile(path string) *File {
	return &File{
		Path: path,
	}
}

// ID returns a unique string that identifies this TemplateSource
func (fs *File) ID() string {
	return "FILE"
}

// Load tries to find a template sub-directory within the filesystem path
func (fs *File) Load(name string) (*model.Template, *derp.Error) {

	directory := fs.Path + "/" + name + "/"

	templateFilename := directory + "template.json"

	data, err := ioutil.ReadFile(templateFilename)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.templateSource.File", "Cannot read file", templateFilename)
	}

	result := model.Template{}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, derp.Wrap(err, "ghost.service.temlpateSource.Filename", "Invalid JSON in template.json", string(data))
	}

	result.Name = name

	// TODO: load separate files from views

	return &result, nil
}
