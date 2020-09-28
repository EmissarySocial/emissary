package service

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestLayout(t *testing.T) {

	layout, err := NewLayout("../layout")

	spew.Dump(layout.Template.DefinedTemplates())
	spew.Dump(err)
}
