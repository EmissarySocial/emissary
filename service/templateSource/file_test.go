package templateSource

import (
	"testing"

	"github.com/benpate/derp"
)

func TestFile(t *testing.T) {

	source := NewFile("test")

	template, err := source.Load("simple")

	derp.Report(err)
	t.Log(template)
}
