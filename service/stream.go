package service

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/nebula"
	"github.com/benpate/rosetta/maps"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	collection          data.Collection
	templateService     *Template
	draftService        *StreamDraft
	attachmentService   *Attachment
	formLibrary         *form.Library
	contentLibrary      *nebula.Library
	streamUpdateChannel chan model.Stream
}

// NewStream returns a fully populated Stream service.
func NewStream(collection data.Collection, templateService *Template, draftService *StreamDraft, attachmentService *Attachment, formLibrary *form.Library, contentLibrary *nebula.Library, streamUpdateChannel chan model.Stream) Stream {

	return Stream{
		collection:          collection,
		templateService:     templateService,
		draftService:        draftService,
		attachmentService:   attachmentService,
		formLibrary:         formLibrary,
		contentLibrary:      contentLibrary,
		streamUpdateChannel: streamUpdateChannel,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// New returns a new stream that uses the named template.
func (service *Stream) New(parent *model.Stream, templateID string) (model.Stream, error) {

	const location = "service.Stream.New"

	template, err := service.templateService.Load(templateID)

	if err != nil {
		return model.Stream{}, derp.Wrap(err, location, "Invalid template", templateID)
	}

	result := model.NewStream()
	result.TemplateID = templateID
	result.ParentID = parent.StreamID
	result.ParentIDs = append(parent.ParentIDs, parent.StreamID)
	result.AsFeature = template.AsFeature

	// TODO: User template schema to set default values in the new stream.

	return result, nil
}

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
	defaultRoles := template.Default().AllowedRoles(stream.StateID)
	stream.DefaultAllow = stream.Permissions.Groups(defaultRoles...)

	// RULE: Copy AsFeature flag from Template into Stream
	stream.AsFeature = template.AsFeature

	// RULE: Calculate rank
	if stream.Rank == 0 {
		maxRank, err := service.MaxRank(context.TODO(), stream.ParentID)

		if err != nil {
			return derp.Wrap(err, location, "Error calculating max rank")
		}
		stream.Rank = maxRank
	}

	// RULE: Default Token
	if stream.Token == "" {
		stream.Token = stream.StreamID.Hex()
	}

	// RULE: Sanitize Content
	service.contentLibrary.Validate(&stream.Content)

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
		if err := service.attachmentService.DeleteAllFromStream(stream.StreamID, note); err != nil {
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

func (service *Stream) Debug() maps.Map {
	return maps.Map{
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

// ListAllFeaturesBySelectionAndRank returns all features in the system.  Selected features are
// listed first (according to their sort rank) and unselected features are listed second.  This
// function also returns a slice of strings that contains the templateIds for all selected features.
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

		for index, feature := range features {

			if feature.Value == stream.TemplateID {

				// copy the selected feature into the selected array
				selected = append(selected, feature)
				templateIDs = append(templateIDs, feature.Value)

				// Remove the feature from the list
				features = append(features[:index], features[index+1:]...)
				break
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

// LoadByID returns a single Stream that matches the provided streamID
func (service *Stream) LoadByID(streamID primitive.ObjectID, result *model.Stream) error {
	return service.Load(exp.Equal("_id", streamID), result)
}

// LoadByProductID returns a single Stream with custom data matching the provided productID
func (service *Stream) LoadByProductID(productID string, result *model.Stream) error {
	return service.Load(exp.Equal("data.productId", productID), result)
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

func (service *Stream) LoadWithOptions(criteria exp.Expression, options option.Option, result *model.Stream) error {

	const location = "service.stream.LoadWithOptions"

	it, err := service.List(notDeleted(criteria), options)

	if err != nil {
		return derp.Wrap(err, location, "Error getting iterator")
	}

	for it.Next(result) {
		return nil
	}

	return derp.NewNotFoundError(location, "collection is empty")
}

func (service *Stream) LoadFirstSibling(parentID primitive.ObjectID, result *model.Stream) error {
	return service.LoadWithOptions(exp.Equal("parentId", parentID), option.SortAsc("rank"), result)
}

func (service *Stream) LoadPrevSibling(parentID primitive.ObjectID, rank int, result *model.Stream) error {

	if rank == 0 {
		return service.LoadLastSibling(parentID, result)
	}

	criteria := exp.Equal("parentId", parentID).AndLessThan("rank", rank)
	options := option.SortDesc("rank")

	err := service.LoadWithOptions(criteria, options, result)

	if err == nil {
		return nil
	}

	if derp.NotFound(err) {
		return service.LoadLastSibling(parentID, result)
	}

	return derp.Wrap(err, "service.stream.LoadPreviousSibling", "Error loading Previous Sibling")
}

func (service *Stream) LoadNextSibling(parentID primitive.ObjectID, rank int, result *model.Stream) error {

	criteria := exp.Equal("parentId", parentID).AndGreaterThan("rank", rank)
	options := option.SortAsc("rank")

	err := service.LoadWithOptions(criteria, options, result)

	if err == nil {
		return nil
	}

	if derp.NotFound(err) {
		return service.LoadFirstSibling(parentID, result)
	}

	return derp.Wrap(err, "service.stream.LoadPreviousSibling", "Error loading Previous Sibling")
}

func (service *Stream) LoadLastSibling(parentID primitive.ObjectID, result *model.Stream) error {
	return service.LoadWithOptions(exp.Equal("parentId", parentID), option.SortDesc("rank"), result)
}

func (service *Stream) LoadFirstAttachment(streamID primitive.ObjectID, attachment *model.Attachment) error {

	const location = "service.stream.LoadFirstAttachment"

	attachments, err := service.attachmentService.ListFirstByObjectID(streamID)

	if err != nil {
		return derp.Wrap(err, location, "Error listing attachments")
	}

	for attachments.Next(attachment) {
		return nil
	}

	return derp.NewNotFoundError(location, "No attachments found")
}

// Count returns the number of (non-deleted) records in the Stream collection
func (service *Stream) Count(ctx context.Context, criteria exp.Expression) (int, error) {
	return queries.CountRecords(ctx, service.collection, notDeleted(criteria))
}

// MaxRank returns the maximum rank of all children of a stream
func (service *Stream) MaxRank(ctx context.Context, parentID primitive.ObjectID) (int, error) {
	return queries.MaxRank(ctx, service.collection, parentID)
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

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Stream.DeleteRelatedDuplicate", "Error deleting related duplicate")
	}

	return nil
}

// CreateAndRestoreFeatures guarantees that the provided stream will have features matching the
// provided templateIDs in the correct sort order.  It does guarantee that each feature is unique
// (by deleting potential duplicates) but it does not remove features that do not match this
// list (that's somebody else's job).
func (service *Stream) CreateAndSortFeatures(stream *model.Stream, templateIDs []string) error {

	const location = "service.Stream.CreateOrRestoreFeatures"

	var currentFeatures []model.Stream

	// Get all features (even deleted ones)
	criteria := notDeleted(exp.Equal("parentId", stream.StreamID).AndEqual("asFeature", true))
	if err := service.collection.Query(&currentFeatures, criteria); err != nil {
		return derp.Wrap(err, location, "Error retrieving features from database")
	}

	for index, templateID := range templateIDs {

		found := false

		// Search all features for matching streams.  If found, "restore" the first one, and delete any duplicates
		for _, stream := range currentFeatures {

			if stream.TemplateID == templateID {

				if !found {

					// If this stream has been deleted, then undelete it and all of its ancestors
					if stream.IsDeleted() {

						stream.Journal.DeleteDate = 0
						if err := service.RestoreDeleted(stream.StreamID); err != nil {
							return derp.Wrap(err, location, "Error restoring deleted descendants")
						}
					}

					// Sort the feature into its new position.
					stream.Rank = index

					if err := service.Save(&stream, "Touched by CreateOrRestoreFeatures"); err != nil {
						return derp.Wrap(err, location, "Error touching stream", stream)
					}

					// Remember that we found one match, in case there are duplicates
					found = true
					continue

				}

				// We're here because there's a duplicate feature.
				// This should never happen, but just in case, delete it now.
				if err := service.Delete(&stream, "Touched by CreateOrRestoreFeatures"); err != nil {
					return derp.Wrap(err, location, "Error deleting duplicate feature", stream)
				}
			}
		}

		// If no matching streams were found, then let's create one now.
		if !found {
			newStream, err := service.New(stream, templateID)

			if err != nil {
				return derp.Wrap(err, location, "Error creating new feature")
			}

			// Require that this new stream is a feature.  No hacking...
			if !newStream.AsFeature {
				return derp.NewBadRequestError(location, "Template must be a feature", templateID)
			}

			// Sort this feature correctly.
			newStream.Rank = index

			if err := service.Save(&newStream, "Created by CreateOrRestoreFeatures"); err != nil {
				return derp.Wrap(err, location, "Error creating new feature", stream)
			}
		}
	}

	return nil
}

// DeleteUnusedFeatures soft-deletes all features that are not in the valid template list
func (service *Stream) DeleteUnusedFeatures(streamID primitive.ObjectID, validTemplateIDs []string) error {
	const location = "service.Stream.DeleteUnusedFeatures"

	criteria := exp.Equal("parentId", streamID).AndEqual("asFeature", true)

	if len(validTemplateIDs) > 0 {
		criteria = criteria.AndNotIn("templateId", validTemplateIDs)
	}

	iterator, err := service.List(criteria)

	if err != nil {
		return derp.Wrap(err, location, "Error listing unused features")
	}

	stream := model.NewStream()

	for iterator.Next(&stream) {

		if err := service.Delete(&stream, "Unused feature removed"); err != nil {
			return derp.Wrap(err, location, "Error removing unused feature", stream)
		}

		stream = model.NewStream()
	}

	return nil
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

// RestoreDeleted un-deletes all soft-deleted records underneath a common ancestor.
func (service *Stream) RestoreDeleted(ancestorID primitive.ObjectID) error {

	const location = "service.Stream.RestoreDeleted"

	// Try to list all deleted descendents
	criteria := exp.Equal("parentIds", ancestorID).AndGreaterThan("journal.deleteDate", 0)
	iterator, err := service.collection.List(criteria)

	if err != nil {
		return derp.Wrap(err, location, "Error listing soft-deleted streams")
	}

	// Iterate through all descendents and UnDelete
	stream := model.NewStream()
	for iterator.Next(&stream) {
		stream.Journal.DeleteDate = 0

		if err := service.Save(&stream, "RestoreDeleted stream"); err != nil {
			return derp.Wrap(err, location, "Error restoring deleted stream", stream)
		}

		stream = model.NewStream()
	}

	// No discomfort, no expansion.
	return nil
}

// PurgeDeleted hard deletes all items with the given ancestor that have already been soft-deleted
func (service *Stream) PurgeDeleted(ancestorID primitive.ObjectID) error {

	const location = "service.Stream.PurgeDeleted"

	criteria := exp.Equal("parentIds", ancestorID).AndGreaterThan("journal.deleteDate", 0)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Error purging soft-deleted streams")
	}

	return nil
}

// UpdateStreamsByTemplate pushes every stream that uses a particular template into the streamUpdateChannel.
func (service *Stream) UpdateStreamsByTemplate(templateID string) {

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
