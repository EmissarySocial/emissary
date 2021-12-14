package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
)

type User struct {
	user     model.User
	layout   model.Layout
	actionID string
	Common
}

func NewUser(factory Factory, ctx *steranko.Context, user model.User, actionID string) User {

	layoutService := factory.Layout()
	layout := layoutService.User()

	return User{
		user:     user,
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (u User) ActionID() string {
	return u.actionID
}

// Action returns the model.Action configured into this renderer
func (u User) Action() (model.Action, bool) {
	return u.layout.Action(u.ActionID())
}

// Render generates the string value for this Stream
func (u User) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := u.layout.Action(u.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(u.factory, u, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
		}
	}
	// Success!
	return template.HTML(buffer.String()), nil
}

func (u User) Token() string {
	return u.user.ID()
}

func (u User) object() data.Object {
	return &u.user
}

func (u User) schema() schema.Schema {
	return u.user.Schema()
}

func (u User) common() Common {
	return u.Common
}

func (u User) executeTemplate(wr io.Writer, name string, data interface{}) error {
	return u.layout.HTMLTemplate.ExecuteTemplate(wr, name, data)
}
