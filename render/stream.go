package render

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/benpate/path"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream wraps a model.Stream object and provides functions that make it easy to render an HTML template with it.
type Stream struct {
	factory  Factory           // Factory interface is required for locating other services.
	ctx      *steranko.Context // Contains request context and authentication data.
	template *model.Template   // Template that the Stream uses
	action   *model.Action     // Action being executed
	stream   *model.Stream     // Stream to be displayed
}

// NewStream creates a new object that can generate HTML for a specific stream/view
func NewStream(factory Factory, ctx *steranko.Context, stream *model.Stream, actionID string) (Stream, error) {

	// Try to load the Template associated with this Stream
	templateService := factory.Template()
	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "ghost.render.NewStream", "Cannot load Stream Template", stream)
	}

	// Try to find requested Action
	action, ok := template.Action(actionID)

	if !ok {
		return Stream{}, derp.New(http.StatusBadRequest, "ghost.render.NewStream", "Invalid action")
	}

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	if !action.UserCan(stream, authorization) {
		return Stream{}, derp.New(http.StatusForbidden, "ghost.render.NewStream", "Forbidden")
	}

	// Success.  Populate Stream
	return Stream{
		factory:  factory,
		ctx:      ctx,
		stream:   stream,
		template: template,
		action:   &action,
	}, nil
}

/*******************************************
 * RENDER FUNCTION
 *******************************************/

// Render generates the string value for this Stream
func (w Stream) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if w.IsEmpty() {
		return template.HTML(""), nil
	}

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Stream) View(action string) (template.HTML, error) {

	subStream, err := NewStream(w.factory, w.ctx, w.stream, action)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "ghost.render.Stream.View", "Error creating sub-renderer", action)
	}

	return subStream.Render()
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

// IsEmpty returns TRUE if this renderer is empty
func (w Stream) IsEmpty() bool {
	return w.stream == nil
}

// URL returns the originally requested URL
func (w Stream) URL() string {
	return w.ctx.Request().URL.RequestURI()
}

// StreamID returns the unique ID for the stream being rendered
func (w Stream) StreamID() string {
	return w.stream.StreamID.Hex()
}

// StreamID returns the unique ID for the stream being rendered
func (w Stream) ParentID() string {
	return w.stream.ParentID.Hex()
}

func (w Stream) TopLevelID() string {
	if len(w.stream.ParentIDs) == 0 {
		return w.stream.StreamID.Hex()
	}
	return w.stream.ParentIDs[0].Hex()
}

// StateID returns the current state of the stream being rendered
func (w Stream) StateID() string {
	return w.stream.StateID
}

// TemplateID returns the name of the template being used
func (w Stream) TemplateID() string {
	return w.stream.TemplateID
}

// ActionID returns the name of the action being performed
func (w Stream) ActionID() string {
	return w.action.ActionID
}

// Token returns the unique URL token for the stream being rendered
func (w Stream) Token() string {
	return w.stream.Token
}

// Label returns the Label for the stream being rendered
func (w Stream) Label() string {
	return w.stream.Label
}

// Description returns the description of the stream being rendered
func (w Stream) Description() string {
	return w.stream.Description
}

// Name of the person who created this Stream
func (w Stream) AuthorName() string {
	return w.stream.AuthorName
}

// PhotoURL of the person who created this Stream
func (w Stream) AuthorImage() string {
	return w.stream.AuthorImage
}

// Returns the body content as an HTML template
func (w Stream) Content() template.HTML {
	result := w.stream.Content.View()
	return template.HTML(result)
}

// Returns editable HTML for the body content (requires `editable` flat)
func (w Stream) ContentEditor() template.HTML {
	result := w.stream.Content.Edit("/" + w.Token() + "/draft")
	return template.HTML(result)
}

// PublishDate returns the PublishDate of the stream being rendered
func (w Stream) PublishDate() int64 {
	return w.stream.PublishDate
}

// CreateDate returns the CreateDate of the stream being rendered
func (w Stream) CreateDate() int64 {
	return w.stream.CreateDate
}

// ThumbnailImage returns the thumbnail image URL of the stream being rendered
func (w Stream) ThumbnailImage() string {
	return w.stream.ThumbnailImage
}

// SourceURL returns the thumbnail image URL of the stream being rendered
func (w Stream) SourceURL() string {
	return w.stream.SourceURL
}

// Data returns the custom data map of the stream being rendered
func (w Stream) Data(value string) interface{} {
	return w.stream.Data[value]
}

// Tags returns the tags of the stream being rendered
func (w Stream) Tags() []string {
	return w.stream.Tags
}

// HasParent returns TRUE if the stream being rendered has a parend objec
func (w Stream) HasParent() bool {
	return w.stream.HasParent()
}

func (w Stream) IsCurrentStream() bool {
	return w.stream.Token == list.Head(w.ctx.Path(), "/")
}

func (w Stream) Roles() []string {
	authorization := getAuthorization(w.ctx)
	return w.stream.Roles(authorization)
}

/*******************************************
 * REQUEST INFO
 *******************************************/

// Returns the request method
func (w Stream) Method() string {
	return w.ctx.Request().Method
}

// Returns the designated request parameter
func (w Stream) QueryParam(param string) string {
	return w.ctx.QueryParam(param)
}

// SetQueryParam sets/overwrites a value from the URL query parameters.
func (w Stream) SetQueryParam(param string, value string) string {
	w.ctx.QueryParams().Set(param, value)
	return "" // <- this is a mega-hack, but it works ;)
}

// Action returns the complete information for the action being performed.
func (w Stream) Action() *model.Action {
	return w.action
}

// IsPartialRequest returns TRUE if this is a partial page request from htmx.
func (w Stream) IsPartialRequest() bool {
	return (w.ctx.Request().Header.Get("HX-Request") != "")
}

/*******************************************
 * INDIVIDUAL RELATIONSHIPS
 *******************************************/

// Parent returns a Stream containing the parent of the current stream
func (w Stream) Parent(actionID string) (Stream, error) {

	streamService := w.factory.Stream()
	parent := model.NewStream()

	if err := streamService.LoadParent(w.stream, &parent); err != nil {
		return Stream{}, derp.Wrap(err, "ghost.renderer.Stream.Parent", "Error loading Parent")
	}

	renderer, err := w.newStream(&parent, actionID)

	if err != nil {
		return Stream{}, derp.Wrap(err, "ghost.renderer.Stream.Parent", "Unable to create new Stream")
	}

	return renderer, nil
}

// PrevSibling returns the sibling Stream that immediately preceeds this one, based on the provided sort field
func (w Stream) PrevSibling(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.ParentID),
		exp.LessThan("sort", path.MustGet(w.stream, "sort")),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortDesc(sort)

	return w.makeFirstStream(criteria, sortOption, action), nil
}

// NextSibling returns the sibling Stream that immediately follows this one, based on the provided sort field
func (w Stream) NextSibling(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.ParentID),
		exp.GreaterThan("sort", path.MustGet(w.stream, "sort")),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortAsc(sort)

	return w.makeFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) FirstChild(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.StreamID),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortAsc(sort)

	return w.makeFirstStream(criteria, sortOption, action), nil
}

// FirstChild returns the first child Stream underneath this one, based on the provided sort field
func (w Stream) LastChild(sort string, action string) (Stream, error) {

	criteria := exp.And(
		exp.Equal("parentId", w.stream.StreamID),
		exp.Equal("journal.deleteDate", 0),
	)

	sortOption := option.SortDesc(sort)

	return w.makeFirstStream(criteria, sortOption, action), nil
}

// makeFirstStream scans an iterator for the first stream allowed to this user
func (w Stream) makeFirstStream(criteria exp.Expression, sortOption option.Option, actionID string) Stream {

	streamService := w.factory.Stream()
	iterator, err := streamService.List(criteria, sortOption)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.renderer.Stream.NextSibling", "Database error"))
		return Stream{}
	}

	stream := new(model.Stream)

	for iterator.Next(stream) {
		if result, err := w.newStream(stream, actionID); err == nil {
			return result
		}
		stream = new(model.Stream)
	}

	// Fall through means no streams are valid.  Return an empty renderer instead.
	return Stream{}
}

/*******************************************
 * RESULTSET RELATIONSHIPS
 *******************************************/

// TopLevel returns an array of Streams that have a Zero ParentID
func (w Stream) TopLevel() ([]Stream, error) {
	criteria := exp.Equal("parentId", primitive.NilObjectID)
	resultSet := w.makeResultSet(criteria)
	resultSet.SortField = "rank"
	resultSet.SortDirection = option.SortDirectionAscending
	resultSet.MaxRows = 10
	return resultSet.View()
}

// Siblings returns all Sibling Streams
func (w Stream) Siblings() *ResultSet {
	return w.makeResultSet(exp.Equal("parentId", w.stream.ParentID))
}

// Children returns all child Streams
func (w Stream) Children() *ResultSet {
	return w.makeResultSet(exp.Equal("parentId", w.stream.StreamID))
}

// makeResultSet returns a fully initialized ResultSet
func (w Stream) makeResultSet(criteria exp.Expression) *ResultSet {

	resultSet := NewResultSet(w.factory, w.ctx, criteria)
	resultSet.SortField = w.template.ChildSortType
	resultSet.SortDirection = w.template.ChildSortDirection

	return resultSet
}

/***************************
 * ATTACHMENTS
 **************************/

// Reference to the first file attached to this stream
func (w Stream) Attachment() (model.Attachment, error) {

	var attachment model.Attachment

	attachmentService := w.factory.Attachment()
	iterator, err := attachmentService.ListByStream(w.stream.StreamID)

	if err != nil {
		return attachment, derp.Wrap(err, "ghost.renderer.Stream.Attachments", "Error listing attachments")
	}

	// Just get a single attachment from the Iterator
	iterator.Next(&attachment)

	return attachment, nil
}

// Attachments lists all attachments for this stream.
func (w Stream) Attachments() ([]model.Attachment, error) {

	result := []model.Attachment{}
	attachmentService := w.factory.Attachment()
	iterator, err := attachmentService.ListByStream(w.stream.StreamID)

	if err != nil {
		return result, derp.Wrap(err, "ghost.renderer.Stream.Attachments", "Error listing attachments")
	}

	attachment := new(model.Attachment)
	for iterator.Next(attachment) {
		result = append(result, *attachment)
		attachment = new(model.Attachment)
	}

	return result, nil
}

/***************************
 * ACCESS PERMISSIONS
 **************************/

// IsSignedIn returns TRUE if the user is signed in
func (w Stream) IsAuthenticated() bool {
	return getAuthorization(w.ctx).IsAuthenticated()
}

// CanView returns TRUE if this Request is authorized to access this stream/view
func (w Stream) UserCan(actionID string) bool {

	action, ok := w.template.Action(actionID)

	if !ok {
		return false
	}

	authorization := getAuthorization(w.ctx)

	return action.UserCan(w.stream, authorization)
}

// CanCreate returns all of the templates that can be created underneath
// the current stream.
func (w Stream) CanCreate() []model.Option {

	templateService := w.factory.Template()
	return templateService.ListByContainer(w.template.TemplateID)
}

/***************************
 * MISC HELPER FUNCTIONS
 **************************/

func (w Stream) setAuthor() error {

	userService := w.factory.User()

	authorization := getAuthorization(w.ctx)
	userID := authorization.UserID

	var user model.User

	if err := userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, "ghost.render.Stream.setAuthor", "Error loading User", userID)
	}

	w.stream.AuthorID = user.UserID
	w.stream.AuthorName = user.DisplayName
	w.stream.AuthorImage = user.ImageURL

	return nil
}

// newStream is a shortcut to the NewStream function that reuses the values present in this current Stream
func (w Stream) newStream(stream *model.Stream, actionID string) (Stream, error) {
	return NewStream(w.factory, w.ctx, stream, actionID)
}

// closeModal sets Response header to close a modal on the client and optionally forward to a new location.
func (w *Stream) closeModal(url string) {
	if url == "" {
		w.ctx.Response().Header().Set("HX-Trigger", `"closeModal"`)
	} else {
		w.ctx.Response().Header().Set("HX-Trigger", `{"closeModal":{"nextPage":"`+url+`"}}`)
	}
}
