package templatesource

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

// List returns all Templates produced by this TemplateSource
func (fs *File) List() ([]string, *derp.Error) {

	list, err := ioutil.ReadDir(fs.Path)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.templateSource.File.List", "Unable to list files in filesystem", fs)
	}

	result := make([]string, len(list))

	// Use a separate counter because not all files will be included in the result
	counter := 0
	for _, fileInfo := range list {

		if fileInfo.IsDir() {
			result[counter] = fileInfo.Name()
			counter = counter + 1
		}
	}

	return result, nil
}

// Load tries to find a template sub-directory within the filesystem path
func (fs *File) Load(templateID string) (model.Template, *derp.Error) {

	directory := fs.Path + "/" + templateID + "/"

	templateFilename := directory + "_template.json"

	data, err := ioutil.ReadFile(templateFilename)

	if err != nil {
		return model.Template{}, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Cannot read file", templateFilename)
	}

	result := model.Template{}

	if err := json.Unmarshal(data, &result); err != nil {
		return model.Template{}, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Invalid JSON in template.json", string(data))
	}

	result.TemplateID = templateID

	// Scan for HTML files associated to each view.
	for key, view := range result.Views {

		if view.File == "" {
			continue
		}

		// Read the file from the filesystem
		file, err := ioutil.ReadFile(directory + view.File)

		if err != nil {
			return model.Template{}, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Error reading vies", view.File)
		}

		// Update the view with the HTML template.
		view.HTML = string(file)

		// Write this value back into the map
		result.Views[key] = view
	}

	return result, nil
}
