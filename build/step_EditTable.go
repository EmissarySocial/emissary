package build

import (
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/table"
)

type StepTableEditor struct {
	Path string
	Form form.Element
}

func (step StepTableEditor) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepTableEditor.Get"

	s := builder.schema()
	factory := builder.factory()

	targetURL := step.getTargetURL(builder)
	t := table.New(&s, &step.Form, builder.object(), step.Path, factory.Icons(), targetURL)
	t.UseLookupProvider(builder.lookupProvider())
	t.AllowAll()

	requestURL := builder.request().URL

	if err := t.Draw(requestURL, buffer); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error building HTML"))
	}

	return Continue().WithHeader("Hx-Push-Url", "false")
}

func (step StepTableEditor) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepTableEditor.Post"

	s := builder.schema()
	object := builder.object()

	// Try to get the form post data
	body, err := formdata.Parse(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Failed to bind body", step))
	}

	if edit := builder.QueryParam("edit"); edit != "" {

		// Bounds checking
		editIndex, ok := convert.IntOk(edit, 0)

		if !ok {
			return Halt().WithError(derp.InternalError(location, "Failed to convert edit index", step.Path, edit))
		}

		if editIndex < 0 {
			return Halt().WithError(derp.InternalError(location, "Edit index out of range", step.Path, editIndex))
		}

		// Try to edit the row in the data table
		for _, field := range step.Form.AllElements() {
			path := step.Path + "." + edit + "." + field.Path

			if err := s.Set(object, path, body.Get(field.Path)); err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Error setting value in table", object, field.Path, path, body[field.Path]))
			}
		}

		// Try to delete an existing record
	} else if delete := builder.QueryParam("delete"); delete != "" {

		table, err := s.Get(object, step.Path)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error locating table in data object"))
		}

		// Bounds checking
		deleteIndex, ok := convert.IntOk(delete, 0)

		if !ok {
			return Halt().WithError(derp.InternalError(location, "Failed to convert edit index", step.Path, edit))
		}

		if (deleteIndex < 0) || (deleteIndex >= convert.SliceLength(table)) {
			return Halt().WithError(derp.InternalError(location, "Edit index out of range", step.Path, deleteIndex))
		}

		// Try to find the schema element for this table control
		if ok := builder.schema().Remove(builder.object(), step.Path+"."+delete); !ok {
			return Halt().WithError(derp.InternalError(location, "Failed to remove row from table", step.Path))
		}
	}

	// Once we're done, re-build the table and send it back to the client
	targetURL := step.getTargetURL(builder)

	factory := builder.factory()
	t := table.New(&s, &step.Form, builder.object(), step.Path, factory.Icons(), targetURL)
	t.UseLookupProvider(builder.lookupProvider())
	t.AllowAll()

	if err := t.DrawView(builder.response()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error building HTML"))
	}

	return Continue().WithHeader("Hx-Push-Url", "false")
}

// getTargetURL returns the URL that the table should use for all of its links
func (step StepTableEditor) getTargetURL(builder Builder) string {
	originalPath := builder.request().URL.Path
	actionID := builder.actionID()
	pathSlice := strings.Split(originalPath, "/")

	if pathSlice == nil {
		return ""
	}

	pathSliceLength := len(pathSlice)
	if pathSliceLength < 1 {
		return ""
	}

	pathSlice[pathSliceLength-1] = actionID
	return strings.Join(pathSlice, "/")
}
