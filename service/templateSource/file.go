package templatesource

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/fsnotify/fsnotify"

	minify "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
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
func (fs *File) List() ([]string, error) {

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
func (fs *File) Load(templateID string) (*model.Template, error) {

	result := model.NewTemplate(templateID)

	directory := fs.Path + "/" + templateID

	files, err := ioutil.ReadDir(directory)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Unable to list directory", directory)
	}

	// Make a data structure for all of the views in the system
	views := make(map[string]*template.Template, len(files))

	// Create the minifier
	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	// Views are processed FIRST so we can generate a list of objects to enter into the different States
	for _, file := range files {

		filename := file.Name()
		viewID, extension := list.SplitTail(filename, ".")

		// Try to read the file from the filesystem
		content, err := ioutil.ReadFile(directory + "/" + filename)

		if err != nil {
			derp.Report(derp.Wrap(err, "ghost.service.templateSource.File.Load", "Cannot read file", filename))
			continue
		}

		// Only HTML files beyond this point...
		switch extension {

		case "html":

			contentString := string(content)

			// Try to minify the incoming template... (this should be moved to a different place.)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			// Try to compile the minified content into a Go Template
			result, err := template.New(viewID).Parse(contentString)

			if err != nil {
				derp.Report(derp.Wrap(err, "ghost.model.View.Compiled", "Unable to parse template HTML", contentString))
				continue
			}

			// Save the view in the memory structure for the next run through the file list
			views[viewID] = result

		case "json":

			var temp model.Template

			if err := json.Unmarshal(content, &temp); err != nil {
				return nil, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Invalid JSON configuration file", filename)
			}

			result.Populate(&temp)
		}
	}

	// For each state in this Template
	for stateIndex := range result.States {

		// For each view in this State
		for viewIndex := range result.States[stateIndex].Views {

			// Get the ID of this view
			viewID := (*result).States[stateIndex].Views[viewIndex].ViewID

			// Add (a pointer to) the compiled template into this view.  (Null if missing)
			result.States[stateIndex].Views[viewIndex].Template = views[viewID]
		}
	}

	return result, nil
}

func (fs *File) appendJSON(template *model.Template, data []byte) error {

	var temp model.Template

	if err := json.Unmarshal(data, &temp); err != nil {
		return derp.Wrap(err, "ghost.service.templateSource.File.Load", "Invalid JSON in template.json", string(data))
	}

	template.Populate(&temp)

	return nil
}

/////////////////////////////////////
/// REAL TIME UPDATES

// Watch populates a channel of model.Template objects every time a template is updated.
func (fs *File) Watch(updateChannel chan model.Template) error {

	fmt.Println("templateSource.Watch")
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return derp.Wrap(err, "ghost.service.templateSource", "Could not watch filesystem")
	}

	files, err := ioutil.ReadDir(fs.Path)

	if err != nil {
		return derp.Wrap(err, "ghost.service.templateSource", "Could not read directory", fs.Path)
	}

	for _, file := range files {
		if file.IsDir() {
			if err := watcher.Add(fs.Path + "/" + file.Name()); err != nil {
				return derp.Wrap(err, "ghost.service.templateSource.File", "Error adding watcher on path", fs.Path, file)
			}
		}
	}

	go func() {

		for {
			select {
			case event, ok := <-watcher.Events:

				fmt.Println("templateSource.Watch.receivedEvent")
				if ok {
					if event.Op != fsnotify.Write {
						continue
					}

					templateName := list.Last(list.RemoveLast(event.Name, "/"), "/")

					template, err := fs.Load(templateName)

					if err != nil {
						derp.Report(derp.Wrap(err, "ghost.service.templateSource.File", "Error loading changes to template", event, templateName))
						continue
					}

					fmt.Println("templateSource.Watch.sendingTemplate: " + template.Label)
					updateChannel <- *template
				}

			case err, ok := <-watcher.Errors:

				if ok {
					derp.Report(derp.Wrap(err, "ghost.service.templateSource.File", "Error watching filesystem"))
				}
			}
		}
	}()

	return nil
}
