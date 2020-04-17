package content

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestTabs(t *testing.T) {

	tabs := Tab{
		Format:   TabFormatTabs,
		Sections: []List{List{}, List{}, List{}},
		Labels:   []string{"File", "Edit", "View"},
	}

	spew.Dump(tabs)
	spew.Dump(tabs.HTML())
}
