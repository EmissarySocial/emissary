package service

import (
	"context"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/davecgh/go-spew/spew"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/queries"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	collection            data.Collection
	templateService       *Template
	draftService          *StreamDraft
	attachmentService     *Attachment
	formLibrary           *form.Library
	templateUpdateChannel chan string
	streamUpdateChannel   chan model.Stream
}

// NewStream returns a fully populated Stream service.
func NewStream(collection data.Collection, templateService *Template, draftService *StreamDraft, attachmentService *Attachment, formLibrary *form.Library, templateUpdateChannel chan string, streamUpdateChannel chan model.Stream) Stream {

	return Stream{
		collection:            collection,
		templateService:       templateService,
		draftService:          draftService,
		attachmentService:     attachmentService,
		formLibrary:           formLibrary,
		templateUpdateChannel: templateUpdateChannel,
		streamUpdateChannel:   streamUpdateChannel,
	}
}

/*******************************************
 * REAL-TIME UPDATES
 *******************************************/

// start begins the background watchers used by the Stream Service
func (service *Stream) Watch() {
	for {
		templateID := <-service.templateUpdateChannel
		service.updateStreamsByTemplate(templateID)
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns an iterator containing all of the Streams who match the provided criteria
func (service *Stream) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Stream from the database
func (service *Stream) Load(criteria exp.Expression, stream *model.Stream) error {

	if err := service.collection.Load(notDeleted(criteria), stream); err != nil {
		return derp.Wrap(err, "service.Stream", "Error loading Stream", criteria)
	}

	return nil
}

// Save adds/updates an Stream in the database
func (service *Stream) Save(stream *model.Stream, note string) error {

	const location = "service.Stream"

	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Template", stream.TemplateID)
	}

	// RULE: Calculate "defaultAllow" groups for this stream.
	defaultRoles := template.Default().AllowedRoles(stream)
	stream.DefaultAllow = stream.Permissions.Groups(defaultRoles...)

	// RULE: Copy AsFeature flag from Template into Stream
	stream.AsFeature = template.AsFeature

	// RULE: First Top-Level Item is "home", no other streams can be marked "home"
	if stream.ParentID == primitive.NilObjectID {
		if stream.Rank == 0 {
			stream.Token = "home" // First stream in the list is the "home" page
		} else if stream.Token == "home" {
			stream.Token = "" // No other stream can be marked "home".
		}
	}

	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, location, "Error saving Stream", stream, note)
	}

	// NON-BLOCKING: Notify other processes on this server that the stream has been updated
	go func() {
		service.streamUpdateChannel <- *stream
		// fmt.Println("streamService.Save: sent update update to stream: " + stream.Label)
	}()

	// One milisecond delay prevents overlapping stream.CreateDates.  Deal with it.
	// TODO: There has to be a better way than this...
	time.Sleep(1 * time.Millisecond)

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service *Stream) Delete(stream *model.Stream, note string) error {

	// Delete this Stream
	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream.Delete", "Error deleting Stream", stream, note)
	}

	// Delete related records -- this can happen in the background
	go func() {

		// RULE: Delete all related Children
		if err := service.DeleteChildren(stream, note); err != nil {
			derp.Report(derp.Wrap(err, "service.Stream.Delete", "Error deleting child streams", stream, note))
		}

		// RULE: Delete all related Attachments
		if err := service.attachmentService.DeleteByStream(stream.StreamID, note); err != nil {
			derp.Report(derp.Wrap(err, "service.Stream.Delete", "Error deleting attachments", stream, note))
		}

		// RULE: Delete all related Drafts
		if err := service.draftService.Delete(stream, note); err != nil {
			derp.Report(derp.Wrap(err, "service.Stream.Delete", "Error deleting drafts", stream, note))
		}
	}()

	// Bueno!!
	return nil
}

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Stream) ObjectNew() data.Object {
	result := model.NewStream()
	return &result
}

func (service *Stream) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Stream) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewStream()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Stream) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.Stream), comment)
}

func (service *Stream) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.Stream), comment)
}

func (service *Stream) Debug() datatype.Map {
	return datatype.Map{
		"service": "Stream",
	}
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

// ListTopLevel returns all Streams of type FOLDER at the top of the hierarchy
func (service *Stream) ListTopLevel() (data.Iterator, error) {
	return service.List(
		exp.Equal("parentId", primitive.NilObjectID),
		option.SortAsc("rank"),
	)
}

// ListAncestors returns all Streams that are ancestors of the provided stream.
func (service *Stream) ListAncestors(stream *model.Stream) ([]model.Stream, error) {

	result := make([]model.Stream, len(stream.ParentIDs))
	it, err := service.List(exp.In("_id", stream.ParentIDs))

	if err != nil {
		return result, derp.Wrap(err, "service.Stream.ListAncestors", "Error accessing database", stream)
	}

	temp := model.NewStream()

	for it.Next(&temp) {
		result[len(temp.ParentIDs)] = temp
		temp = model.NewStream()
	}

	return result, nil
}

// ListByParent returns all Streams that match a particular parentID
func (service *Stream) ListByParent(parentID primitive.ObjectID) (data.Iterator, error) {
	return service.List(exp.Equal("parentId", parentID))
}

// ListFeatures returns all Streams that match a particular parentID
func (service *Stream) ListFeatures(streamID primitive.ObjectID) (data.Iterator, error) {
	criteria := exp.Equal("parentId", streamID).AndEqual("asFeature", true)
	return service.List(criteria, option.SortAsc("rank"))
}

func (service *Stream) ListAllFeaturesBySelectionAndRank(streamID primitive.ObjectID) ([]form.OptionCode, []string, error) {

	const location = "service.Stream.ListAllFeaturesBySelectionAndRank"

	streams, err := service.ListFeatures(streamID)

	if err != nil {
		return []form.OptionCode{}, []string{}, derp.Wrap(err, location, "Error getting features for this stream", streamID)
	}

	features := service.templateService.ListFeatures()
	templateIDs := []string{}
	selected := []form.OptionCode{}

	stream := model.NewStream()
	for streams.Next(&stream) {

		for index := range features {

			if features[index].Value == stream.TemplateID {

				// copy the selected feature into the selected array
				selected = append(selected, features[index])
				templateIDs = append(templateIDs, features[index].Value)

				// Remove the feature from the list
				features = append(features[:index], features[index+1:]...)
			}
		}

		stream = model.NewStream()
	}

	selected = append(selected, features...)
	return selected, templateIDs, nil
}

// ListByTemplate returns all Streams that use a particular Template
func (service *Stream) ListByTemplate(template string) (data.Iterator, error) {
	return service.List(exp.Equal("templateId", template))
}

// LoadByToken returns a single Stream that matches a particular Token
func (service *Stream) LoadByToken(token string, result *model.Stream) error {

	// If the token looks like an ObjectID, then try Load by ID first.
	if streamID, err := primitive.ObjectIDFromHex(token); err == nil {
		if err := service.LoadByID(streamID, result); err == nil {
			return nil
		}
	}

	// Default to Load by Token
	return service.Load(exp.Equal("token", token), result)
}

// LoadByID returns a single Stream that matches a particular StreamID
func (service *Stream) LoadByID(streamID primitive.ObjectID, result *model.Stream) error {
	return service.Load(exp.Equal("_id", streamID), result)
}

// LoadBySource locates a single stream that matches the provided SourceURL
func (service *Stream) LoadBySource(parentStreamID primitive.ObjectID, sourceURL string, result *model.Stream) error {

	criteria := exp.
		Equal("parentId", parentStreamID).
		AndEqual("sourceUrl", sourceURL)

	return service.Load(criteria, result)
}

// LoadParent returns the Stream that is the parent of the provided Stream
func (service *Stream) LoadParent(stream *model.Stream, parent *model.Stream) error {

	if !stream.HasParent() {
		return derp.NewNotFoundError("service.Stream.LoadParent", "Stream does not have a parent")
	}

	if err := service.LoadByID(stream.ParentID, parent); err != nil {
		return derp.Wrap(err, "service.stream.LoadParent", "Error loading parent", stream)
	}

	return nil
}

// LoadTopLevelByID locates a single stream in the top level of the site hierarchy
func (service *Stream) LoadTopLevelByID(streamID primitive.ObjectID, result *model.Stream) error {

	criteria := exp.
		Equal("_id", streamID).
		AndEqual("parentId", primitive.NilObjectID)

	return service.Load(criteria, result)
}

// Count returns the number of (non-deleted) records in the Stream collection
func (service *Stream) Count(ctx context.Context, criteria exp.Expression) (int, error) {
	return queries.CountRecords(ctx, service.collection, notDeleted(criteria))
}

/*******************************************
 * CUSTOM ACTIONS
 *******************************************/

// DeleteChildren removes all child streams from the provided stream (virtual delete)
func (service *Stream) DeleteChildren(stream *model.Stream, note string) error {

	var child model.Stream
	it, err := service.ListByParent(stream.StreamID)

	if err != nil {
		return derp.Wrap(err, "service.Stream.Delete", "Error listing child streams", stream)
	}

	for it.Next(&child) {
		if err := service.Delete(&child, note); err != nil {
			return derp.Wrap(err, "service.Stream.Delete", "Error deleting child stream", child)
		}
	}

	return nil
}

// Delete RelatedDuplicate hard deletes any inbox/outbox streams that point to the same original.
func (service *Stream) DeleteRelatedDuplicate(parentID primitive.ObjectID, originalStreamID primitive.ObjectID) error {

	criteria := exp.Equal("parentId", parentID).AndEqual("data.originalStreamId", originalStreamID)

	spew.Dump("DeleteRelatedDuplicate", criteria)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Stream.DeleteRelatedDuplicate", "Error deleting related duplicate")
	}

	return nil
}

// updateStreamsByTemplate pushes every stream that uses a particular template into the streamUpdateChannel.
func (service *Stream) updateStreamsByTemplate(templateID string) {

	iterator, err := service.ListByTemplate(templateID)

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Realtime", "Error Listing Streams for Template", templateID))
		return
	}

	stream := model.NewStream()

	for iterator.Next(&stream) {
		service.streamUpdateChannel <- stream
		stream = model.NewStream()
	}
}

// CreatePersonalStream generates a hidden stream that is tightly linked to a specific user.
// Used to create inbox/outbox streams
func (service *Stream) CreatePersonalStream(user *model.User, templateID string) (primitive.ObjectID, error) {

	stream := model.NewStream()
	stream.TemplateID = templateID
	stream.ParentID = user.UserID
	stream.AuthorID = user.UserID
	stream.Permissions = model.NewPermissions()
	stream.Permissions.Assign("myself", user.UserID)

	err := service.Save(&stream, "auto: create inbox")

	return stream.StreamID, err
}
