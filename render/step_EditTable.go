package render

import (
	"io"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/table"
	"github.com/labstack/echo/v4"
)

type StepTableEditor struct {
	Path string
	Form form.Element
}

func (step StepTableEditor) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepTableEditor.Get"
	var err error

	s := renderer.schema()
	factory := renderer.factory()

	targetURL := step.getTargetURL(renderer)
	t := table.New(&s, &step.Form, renderer.object(), step.Path, factory.Icons(), targetURL)
	t.UseLookupProvider(factory.LookupProvider())
	t.AllowAll()

	if editRow, ok := convert.IntOk(renderer.QueryParam("edit"), 0); ok {
		err = t.DrawEdit(editRow, buffer)
	} else if add := renderer.QueryParam("add"); add != "" {
		err = t.DrawAdd(buffer)
	} else {
		err = t.DrawView(buffer)
	}

	return derp.Wrap(err, location, "Error drawing table", step.Path)
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
				return derp.Wrap(err, location, "Error setting value in table", object, field.Path, path, body[field.Path])
			}
		}

		// Try to delete an existing record
	} else if delete := renderer.QueryParam("delete"); delete != "" {

		table, err := s.Get(object, step.Path)

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
		if ok := renderer.schema().Remove(renderer.object(), step.Path+"."+delete); !ok {
			return derp.NewInternalError(location, "Failed to remove row from table", step.Path)
		}
	}

	// Once we're done, re-render the table and send it back to the client
	targetURL := step.getTargetURL(renderer)

	factory := renderer.factory()
	t := table.New(&s, &step.Form, renderer.object(), step.Path, factory.Icons(), targetURL)
	t.UseLookupProvider(factory.LookupProvider())
	t.AllowAll()

	if err := t.DrawView(renderer.context().Response().Writer); err != nil {
		return derp.Wrap(err, location, "Error building HTML")
	}

	return nil
}

// getTargetURL returns the URL that the table should use for all of its links
func (step StepTableEditor) getTargetURL(renderer Renderer) string {
	originalPath := renderer.context().Request().URL.Path
	actionID := renderer.ActionID()
	pathSlice := strings.Split(originalPath, "/")
	pathSlice[len(pathSlice)-1] = actionID
	return strings.Join(pathSlice, "/")
}
