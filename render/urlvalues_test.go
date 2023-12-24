package render

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

// TestSetRawQuery is an experiment to see how templates work
// when setting individual query values
func TestSetRawQuery_Fails(t *testing.T) {

	temp := template.Must(template.New("test").Parse(`<a href="https://someserver.com?city=Denver&{{.}}">Link</a>`))
	rawQuery := "name=John&age=30&city=New%20York&city=San%20Diego&city=Los%20Angeles"

	t.Log(runTemplate(temp, rawQuery))
}

func TestSetRawQuery(t *testing.T) {

	temp := template.Must(template.New("test").Parse(`<a href="https://someserver.com?city=Denver&{{.}}">Link</a>`))
	rawQuery := template.URL("name=John&age=30&city=New%20York&city=San%20Diego&city=Los%20Angeles")

	spew.Dump(runTemplate(temp, rawQuery))
}

// runTemplate executes a template, and returns the result as a string
func runTemplate(t *template.Template, value any) string {
	var result bytes.Buffer
	_ = t.Execute(&result, value)
	return result.String()
}
