package templateSource

import (
	"encoding/json"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/davecgh/go-spew/spew"
	"github.com/fsnotify/fsnotify"
)

// File is a TemplateSource adapter that can load/save Templates from/to the local filesytem.
type File struct {
	Path string
}

// NewFile returns a fully initialized File adapter for loading/saving Templates
func NewFile(path string) *File {
	return &File{
		Path: path,
	}
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

// Watch populates a channel of model.Template objects every time a template is updated.
func (fs *File) Watch(results chan model.Template) {

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.templateSource", "Could not watch filesystem"))
	}

	files, err := ioutil.ReadDir(fs.Path)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.templateSource", "Could not read directory", fs.Path))
	}

	go func() {

		for {
			select {
			case event, ok := <-watcher.Events:

				if ok {

					if _, err := ioutil.ReadDir(event.Name); err == nil {

						fileName := list.Last(event.Name, "/")
						template, err := fs.Load(fileName)

						if err != nil {
							derp.Report(derp.Wrap(err, "ghost.service.templateSource.File", "Error loading changes to template", event, fileName))
							continue
						}

						spew.Dump("Template Watcher.  Updating", template)
						results <- template
					}
				}

			case err, ok := <-watcher.Errors:
				derp.Report(derp.Wrap(err, "ghost.service.templateSource.File", "Error watching filesystem"))
				spew.Dump(err)
				spew.Dump(ok)
			}
		}
	}()

	for _, file := range files {
		if file.IsDir() {
			if err := watcher.Add(fs.Path + "/" + file.Name()); err != nil {
				derp.Report(derp.Wrap(err, "ghost.service.templateSource.File", "Error adding watcher on path", fs.Path, file))
			}
		}
	}
}
