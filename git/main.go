package git

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

func doWork() {

	// Clones the given repository in memory, creating the remote, the local
	// branches and fetching the objects, exactly as:
	// Info("git clone https://github.com/go-git/go-billy")

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: "https://github.com/benpate/ghost-packages",
	})

	// Gets the HEAD history from HEAD, just like this command:
	// Info("git log")

	// ... retrieves the branch pointed by HEAD
	ref, err := r.Head()
	CheckIfError(err)

	commit, err := r.CommitObject(ref.Hash())
	CheckIfError(err)

	// ... retrieve the tree from the commit
	tree, err := commit.Tree()
	CheckIfError(err)

	// ... get the files iterator and print the file
	tree.Files().ForEach(func(f *object.File) error {
		spew.Dump(f)
		return nil
	})

	/*
		// ... just iterates over the commits, printing it
		err = cIter.ForEach(func(c *object.Commit) error {
			fmt.Println(c)
			return nil
		})
		CheckIfError(err)
	*/
}

func CheckIfError(err error) {

	if err == nil {
		return
	}

	panic(err)
}
