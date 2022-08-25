package render

import (
	"io"
	"reflect"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"github.com/labstack/echo/v4"
)

type StepTableEditor struct {
	Path string
	Form form.Element
}

func (step StepTableEditor) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepTableEditor.Get"

	s := renderer.schema()
	targetURL := "/" + renderer.Token() + "/" + renderer.ActionID()

	editRow, ok := convert.IntOk(renderer.QueryParam("edit"), 0)

	if !ok {
		editRow = -1
	}

	if err := step.drawTable(&s, renderer.object(), targetURL, editRow, buffer); err != nil {
		return derp.Wrap(err, location, "Error building HTML")
	}

	return nil
}

func (step StepTableEditor) UseGlobalWrapper() bool {
	return true
}

func (step StepTableEditor) Post(renderer Renderer) error {

	const location = "render.StepTableEditor.Post"

	s := renderer.schema()
	object := renderer.object()

	// Try to get the form post data
	body := make(maps.Map)

	if err := (&echo.DefaultBinder{}).BindBody(renderer.context(), &body); err != nil {
		return derp.Wrap(err, location, "Failed to bind body", step)
	}

	if edit := renderer.QueryParam("edit"); edit != "" {

		// Bounds checking
		editIndex, ok := convert.IntOk(edit, 0)

		if !ok {
			return derp.NewInternalError(location, "Failed to convert edit index", step.Path, edit)
		}

		if editIndex < 0 {
			return derp.NewInternalError(location, "Edit index out of range", step.Path, editIndex)
		}

		// Try to edit the row in the data table
		for _, field := range step.Form.AllElements() {
			path := step.Path + "." + edit + "." + field.Path

			if err := s.Set(object, path, body[field.Path]); err != nil {
				return derp.Wrap(err, location, "Error setting value in table")
			}
		}

		// Try to delete an existing record
	} else if delete := renderer.QueryParam("delete"); delete != "" {

		table, _, err := s.Get(object, step.Path)

		if err != nil {
			return derp.Wrap(err, location, "Error locating table in data object")
		}

		// Bounds checking
		deleteIndex, ok := convert.IntOk(delete, 0)

		if !ok {
			return derp.NewInternalError(location, "Failed to convert edit index", step.Path, edit)
		}

		if (deleteIndex < 0) || (deleteIndex >= convert.SliceLength(table)) {
			return derp.NewInternalError(location, "Edit index out of range", step.Path, deleteIndex)
		}

		// Try to find the schema element for this table control
		if err := renderer.schema().Remove(renderer.object(), step.Path+"."+delete); err != nil {
			return derp.Wrap(err, location, "Failed to remove row from table", step.Path)
		}
	}

	// Once we're done, re-render the table and send it back to the client
	targetURL := "/" + renderer.Token() + "/" + renderer.ActionID()

	if err := step.drawTable(&s, renderer.object(), targetURL, -1, renderer.context().Response().Writer); err != nil {
		return derp.Wrap(err, location, "Error building HTML")
	}

	return nil
}

func (step StepTableEditor) drawTable(s *schema.Schema, object any, targetURL string, editRow int, buffer io.Writer) error {

	const location = "render.StepTableEditor.view"

	// Try to locate (and validate) that we have a usable schema for a table
	value, tableElement, err := s.Get(object, step.Path)

	if err != nil {
		return derp.Wrap(err, location, "Failed to locate table schema", step.Path)
	}

	arrayElement, ok := tableElement.(schema.Array)

	if !ok {
		return derp.NewInternalError(location, "Table schema must be an array", step.Path, tableElement)
	}

	arraySchema := schema.New(arrayElement)

	// Begin rendering the table
	b := html.New()

	b.Div().Role("table").
		Data("hx-target", "this").
		Data("hx-swap", "outerHTML").
		Data("hx-push_url", "false")

	b.Div().Role("row")
	for _, field := range step.Form.Children {
		b.Div().Role("columnheader").InnerHTML(field.Label).Close()
	}
	b.Div().Role("columnheader").Close() // This will be the "actions" column

	b.Close() // .row

	length := convert.SliceLength(value)
	valueOf := reflect.ValueOf(value)

	for index := 0; index < length; index++ {

		indexString := convert.String(index)
		row, rowElement, err := arraySchema.Get(valueOf, indexString)

		if err != nil {
			return derp.Wrap(err, location, "Failed to locate row schema", step.Path, index)
		}

		rowSchema := schema.New(rowElement)

		if index == editRow {

			if err := step.drawEditRow(&rowSchema, row, targetURL, indexString, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Failed to draw edit row", step.Path, index)
			}

		} else {

			if err := step.drawViewRow(&rowSchema, row, targetURL, indexString, b.SubTree()); err != nil {
				return derp.Wrap(err, location, "Failed to draw row", step.Path, index)
			}
		}
	}

	// If we're not editing an existing row, then let users add a new row
	if editRow == -1 {
		step.drawAddRow(s, targetURL, length, b.SubTree())
	}

	b.CloseAll()

	buffer.Write(b.Bytes())
	return nil
}

func (step StepTableEditor) drawAddRow(s *schema.Schema, targetURL string, index int, b *html.Builder) {

	b.Form("", "").
		Data("hx-post", targetURL+"?edit="+convert.String(index)).
		Role("row").
		Class("add")

	// b.Div().Role("row").Class("add")

	for _, field := range step.Form.Children {
		b.Div().Role("cell")
		field.WriteHTML(s, nil, nil, b.SubTree())
		b.Close() // .cell
	}

	b.Div().Role("cell").Class("align-right")
	b.Button().Class("primary", "text-sm").InnerHTML("Add").Close()

	b.Close() // .cell
	// b.Close() // .row
	b.Close() // form
}

func (step StepTableEditor) drawEditRow(rowSchema *schema.Schema, row any, targetURL string, index string, b *html.Builder) error {

	b.Form("", "").
		Role("row").
		Data("hx-post", targetURL+"?edit="+index)

	// b.Div().Role("row").Class("edit")

	for _, field := range step.Form.Children {
		b.Div().Role("cell")
		field.WriteHTML(rowSchema, nil, row, b.SubTree())
		b.Close() // .cell
	}

	// Write actions column
	b.Div().Role("cell").Class("align-right")
	b.Button().
		Type("submit").
		Class("primary", "text-sm").
		InnerHTML("Save").Close()

	b.Button().
		Type("button").
		Class("btn", "text-sm").
		Data("hx-get", targetURL).
		InnerHTML("Cancel").Close()
	b.Close() // .cell
	// b.Close() // .row
	b.Close() // form

	return nil
}

func (step StepTableEditor) drawViewRow(rowSchema *schema.Schema, row any, targetURL string, index string, b *html.Builder) error {

	b.Div().Role("row").
		Data("hx-get", targetURL+"?edit="+index).
		Data("hx-trigger", "click")

	for _, field := range step.Form.Children {
		cellValue, _, err := rowSchema.Get(row, field.Path)

		if err != nil {
			return derp.Wrap(err, "render.StepTableEditor.drawEditRow", "Failed to locate cell schema", step.Path, index, field.Path)
		}

		b.Div().Role("cell").InnerHTML(convert.String(cellValue)).Close()
	}

	b.Div().Role("cell").Class("align-right")

	b.A("").
		Role("button").
		Data("hx-get", targetURL+"?edit="+index)

	b.I("ti", "ti-edit").Close()
	b.WriteString("Edit")
	b.Close()

	b.Space()

	b.A("").
		Role("button").
		Class("text-red").
		Data("hx-confirm", "Are you sure you want to delete this row?").
		Data("hx-post", targetURL+"?delete="+index)

	b.I("ti", "ti-trash").Close()
	b.WriteString("Delete")
	b.Close()

	b.Close() // .cell
	b.Close() // .row

	return nil
}
