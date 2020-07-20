package templatesource

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestConnect(t *testing.T) {

	g := NewGit("https://github.com/benpate/ghost-packages")

	err := g.Load()

	spew.Dump(g)
	spew.Dump(err)

	// t.Fail()
}
