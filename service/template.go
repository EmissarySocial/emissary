package service

import (
	"github.com/benpate/data"
	"github.com/davecgh/go-spew/spew"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	factory Factory
	session data.Session
}

// HTML retrieves the appropriate template for the provided object, and merges the object's data into the template.
func (service Template) HTML(object interface{}) string {

	return `<html><pre>` + spew.Sdump(object) + `</pre></html>`
}
