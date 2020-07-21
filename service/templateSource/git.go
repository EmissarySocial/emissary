package templatesource

import (
	"bytes"
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Git represents the Git connector service, that can populate Packages and Templates from a Git repository
type Git struct {
	URL        string
	Package    model.Package
	Templates  map[string]*model.Template
	BlockLists map[string]*model.BlockList
}

// NewGit returns a fully initialized Git connector service
func NewGit(url string) Git {

	return Git{
		URL:        url,
		Package:    model.Package{},
		Templates:  map[string]*model.Template{},
		BlockLists: map[string]*model.BlockList{},
	}
}

func (g *Git) Register(_ TemplateService) {
	// Nothing to do right now.
}

// List tries to return all Templates produced by this TemplateSource
func (g *Git) List() ([]string, *derp.Error) {
	return nil, derp.New(500, "ghost.service.templateSource.Git.List", "Unimplemented")
}

// Load retrieves a package from a remote Git repository
func (g *Git) Load() error {
	// Clones the given repository in memory, creating the remote, the local
	// branches and fetching the objects, exactly as:
	// Info("git clone https://github.com/go-git/go-billy")

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: g.URL,
	})

	if err != nil {
		return derp.New(500, "service.connector.git.Load", "Error connecting to Git server", g.URL, err)
	}

	// Gets the HEAD history from HEAD, just like this command:
	// Info("git log")

	// ... retrieves the branch pointed by HEAD
	ref, err := r.Head()

	if err != nil {
		return derp.New(500, "service.connector.git.Load", "Error retrieving HEAD from Git server", g.URL, err)
	}

	commit, err := r.CommitObject(ref.Hash())

	if err != nil {
		return derp.New(500, "service.connector.git.Load", "Error retrieving CommitObject from Git server", g.URL, err)
	}

	// ... retrieve the tree from the commit
	tree, err := commit.Tree()

	if err != nil {
		return derp.New(500, "service.connector.git.Load", "Error retrieving Tree from Git server", g.URL, err)
	}

	// ... get the files iterator and print the file
	err = tree.Files().ForEach(g.ParseFile)

	if err != nil {
		return derp.New(500, "service.connector.git.Load", "Error iterating over file directory from Git server", g.URL, err)
	}

	return nil
}

// ParseFile knows how to "do the right thing" with a given Git file reference
func (g *Git) ParseFile(f *object.File) error {

	switch f.Name {

	case "README.md":
		g.parseReadMe(f)
		return nil

	case "package.json":
		g.parsePackage(f)
		return nil
	}

	folder, filename := list.Split(f.Name, "/")

	switch folder {

	case "template":

		folder, filename := list.Split(filename, "/")

		switch filename {

		case "content.html":
			g.parseHTMLTemplate(folder, f)
			return nil

		case "schema.json":
			g.parseJSONSchema(folder, f)
			return nil

		case "form.json":
			g.parseJSONForm(folder, f)
			return nil
		}

	case "blocklist":
		g.parseBlockList(folder, f)
		return nil
	}

	// Fall through to here means this file is not recognized, and will be ignored.
	return nil
}

func (g *Git) parseReadMe(f *object.File) error {

	readme, err := g.fileContents(f)

	if err != nil {
		return derp.Wrap(err, "service.connector.git.parseReadMe", "Error getting file contents")
	}

	g.Package.ReadMe = string(readme)

	return nil
}

func (g *Git) parsePackage(f *object.File) error {

	contents, err := g.fileContents(f)

	if err != nil {
		return derp.Wrap(err, "service.connector.git.parsePackage", "Error getting file contents")
	}

	if err := json.Unmarshal(contents, &g.Package); err != nil {
		return derp.New(500, "service.connector.git.parsePackage", "Error parsing JSON", f.Name, err)
	}

	return nil
}

func (g *Git) parseHTMLTemplate(name string, f *object.File) error {

	/*
		contents, err := g.fileContents(f)

		if err != nil {
			return derp.Wrap(err, "service.connector.git.parseHTMLTemplate", "Error getting file contents")
		}

		// Get (or create) the Template for this name, and load it with the contents.
		t := g.getTemplateByName(name)
		// t.Content = string(contents)

		// TODO: consider minifying with https://github.com/tdewolff/minify
	*/
	return nil
}

func (g *Git) parseJSONSchema(name string, f *object.File) error {

	/*
		contents, err := g.fileContents(f)

		if err != nil {
			return derp.Wrap(err, "service.connector.git.parseHTMLTemplate", "Error getting file contents")
		}

		schema := jsonschema.Schema{}

		if err := json.Unmarshal(contents, &schema); err != nil {
			return derp.New(500, "Cannot unmarshal JSON schema", string(contents), err)
		}

		t := g.getTemplateByName(name)
		t.Schema = schema
	*/
	return nil
}

func (g *Git) parseJSONForm(name string, f *object.File) error {
	return nil
}

func (g *Git) parseBlockList(name string, f *object.File) error {
	return nil
}

func (g *Git) getTemplateByName(name string) *model.Template {

	if _, ok := g.Templates[name]; !ok {
		g.Templates[name] = &model.Template{}
	}

	return g.Templates[name]
}

func (g *Git) fileContents(f *object.File) ([]byte, error) {

	var result bytes.Buffer

	reader, err := f.Blob.Reader()

	if err != nil {
		return nil, derp.New(500, "service.connector.git.fileContents", "Error getting reader from git file", f.Name, err)
	}

	result.ReadFrom(reader)

	return result.Bytes(), nil
}
