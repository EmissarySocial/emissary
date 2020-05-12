package git

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/davecgh/go-spew/spew"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Git represents the Git connector service, that can populate Packages and Templates from a Git repository
type Git struct {
	URL        string
	Package    model.Package
	Templates  map[string]model.Template
	BlockLists map[string]model.BlockList
}

// New returns a fully initialized Git connector service
func New(url string) Git {

	return Git{
		URL:        url,
		Package:    model.Package{},
		Templates:  map[string]model.Template{},
		BlockLists: map[string]model.BlockList{},
	}

}

// Load retrieves a package from a remote Git repository
func (g *Git) Load() *derp.Error {
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
		spew.Dump("HERE?", err.(*derp.Error))
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

	if strings.HasPrefix(f.Name, "template") {
		g.parseTemplate(f)
		return nil
	}

	if strings.HasPrefix(f.Name, "blocklist") {
		g.parseBlockList(f)
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

func (g *Git) parsePackage(f *object.File) *derp.Error {

	contents, err := g.fileContents(f)

	if err != nil {
		return derp.Wrap(err, "service.connector.git.parsePackage", "Error getting file contents")
	}

	if err := json.Unmarshal(contents, &g.Package); err != nil {
		return derp.New(500, "service.connector.git.parsePackage", "Error parsing JSON", f.Name, err)
	}

	spew.Dump(string(contents))
	spew.Dump(g.Package)

	return nil
}

func (g *Git) parseTemplate(f *object.File) *derp.Error {
	return nil
}

func (g *Git) parseBlockList(f *object.File) *derp.Error {
	return nil
}

func (g *Git) fileContents(f *object.File) ([]byte, *derp.Error) {

	var result bytes.Buffer

	reader, err := f.Blob.Reader()

	if err != nil {
		return nil, derp.New(500, "service.connector.git.fileContents", "Error getting reader from git file", f.Name, err)
	}

	result.ReadFrom(reader)

	return result.Bytes(), nil
}
