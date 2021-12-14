package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
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
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (u *User) GetPath(p path.Path) (interface{}, error) {
	return u.user.GetPath(p)
}

func (u *User) SetPath(p path.Path, value interface{}) error {
	return u.user.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (user User) ActionID() string {
	return user.actionID
}

// Action returns the model.Action configured into this renderer
func (user User) Action() (model.Action, bool) {
	return user.layout.Action(user.ActionID())
}

// Render generates the string value for this Stream
func (user User) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := user.layout.Action(user.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(user.factory, &user, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
		}
	}
	// Success!
	return template.HTML(buffer.String()), nil
}

func (user User) Token() string {
	return user.user.ID()
}

func (user User) object() data.Object {
	return &user.user
}

func (user User) schema() schema.Schema {
	return user.user.Schema()
}

func (user User) common() Common {
	return user.Common
}

func (user User) executeTemplate(writer io.Writer, name string, data interface{}) error {
	return user.layout.HTMLTemplate.ExecuteTemplate(writer, name, data)
}
