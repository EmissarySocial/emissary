package service

import (
	"context"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Block defines a service that manages all content blocks created and imported by Users.
type Block struct {
	collection      data.Collection
	followerService *Follower
	userService     *User

	queue *queue.Queue
}

// NewBlock returns a fully initialized Block service
func NewBlock() Block {
	return Block{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Block) Refresh(collection data.Collection, followerService *Follower, userService *User, queue *queue.Queue) {
	service.collection = collection
	service.followerService = followerService
	service.userService = userService
	service.queue = queue
}

// Close stops any background processes controlled by this service
func (service *Block) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// Query returns an slice of allthe Blocks that match the provided criteria
func (service *Block) Query(criteria exp.Expression, options ...option.Option) ([]model.Block, error) {
	result := make([]model.Block, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Blocks that match the provided criteria
func (service *Block) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Channel returns a channel that will stream all of the Blocks that match the provided criteria
func (service *Block) Channel(criteria exp.Expression, options ...option.Option) (<-chan model.Block, error) {
	it, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Block.Channel", "Error creating iterator", criteria, options)
	}

	return data.Channel(it, model.NewBlock), nil
}

// Load retrieves an Block from the database
func (service *Block) Load(criteria exp.Expression, block *model.Block) error {

	if err := service.collection.Load(notDeleted(criteria), block); err != nil {
		return derp.Wrap(err, "service.Block.Load", "Error loading Block", criteria)
	}

	return nil
}

// Save adds/updates an Block in the database
func (service *Block) Save(block *model.Block, note string) error {

	if block.IsNew() {
		var err error

		switch block.Type {

		case model.BlockTypeActor:
			err = service.ValidateNewActor(block)

		case model.BlockTypeDomain:
			err = service.ValidateNewDomain(block)

		case model.BlockTypeContent:
			err = service.ValidateNewContent(block)
		}

		if err != nil {
			return derp.Wrap(err, "service.Block.Save", "Error validating new Block", block)
		}
	}

	// Clean the value before saving
	if err := service.Schema().Clean(block); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error cleaning Block", block)
	}

	// Save the block to the database
	if err := service.collection.Save(block, note); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error saving Block", block, note)
	}

	// RULE: Publish changes when the block is first shared publicly
	if block.IsActive && block.IsPublic && (block.PublishDate == 0) {
		if err := service.publish(block); err != nil {
			return derp.Wrap(err, "service.Block.Save", "Error publishing Block", block)
		}
	}

	// RULE: Unpublish changes when the block is no longer shared publicly
	if (!block.IsPublic || !block.IsActive) && (block.PublishDate > 0) {
		if err := service.unpublish(block, true); err != nil {
			return derp.Wrap(err, "service.Block.Save", "Error unpublishing Block", block)
		}
	}

	// Recalculate the block count for this user
	go service.userService.CalcBlockCount(block.UserID)

	// TODO: HIGH: Remove matching followers...

	return nil
}

// Delete removes an Block from the database (virtual delete)
func (service *Block) Delete(block *model.Block, note string) error {

	// Delete this Block
	if err := service.collection.Delete(block, note); err != nil {
		return derp.Wrap(err, "service.Block.Delete", "Error deleting Block", block, note)
	}

	if block.IsPublic {
		if err := service.unpublish(block, false); err != nil {
			derp.Report(derp.Wrap(err, "service.Block.Delete", "Error unpublishing Block", block)) // Fail loudly, but don't block.
		}
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Block) ObjectType() string {
	return "Block"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *Block) ObjectNew() data.Object {
	result := model.NewBlock()
	return &result
}

func (service *Block) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Block); ok {
		return mention.BlockID
	}

	return primitive.NilObjectID
}

func (service *Block) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Block) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Block) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewBlock()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Block) ObjectSave(object data.Object, comment string) error {
	if block, ok := object.(*model.Block); ok {
		return service.Save(block, comment)
	}
	return derp.NewInternalError("service.Block.ObjectSave", "Invalid Object Type", object)
}

func (service *Block) ObjectDelete(object data.Object, comment string) error {
	if block, ok := object.(*model.Block); ok {
		return service.Delete(block, comment)
	}
	return derp.NewInternalError("service.Block.ObjectDelete", "Invalid Object Type", object)
}

func (service *Block) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Block", "Not Authorized")
}

func (service *Block) Schema() schema.Schema {
	return schema.New(model.BlockSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Block) LoadByID(userID primitive.ObjectID, blockID primitive.ObjectID, block *model.Block) error {

	criteria := exp.Equal("_id", blockID).
		And(service.byUserID(userID))

	return service.Load(criteria, block)
}

func (service *Block) LoadByToken(userID primitive.ObjectID, token string, block *model.Block) error {
	blockID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Block.LoadByToken", "Error converting token to ObjectID", token)
	}

	criteria := exp.Equal("_id", blockID).AndEqual("userId", userID)

	return service.Load(criteria, block)
}

func (service *Block) CountByType(userID primitive.ObjectID, blockType string) (int, error) {
	return queries.CountBlocksByType(context.Background(), service.collection, userID, blockType)
}

func (service *Block) QueryActiveByUser(userID primitive.ObjectID) ([]model.Block, error) {
	return service.Query(
		exp.Equal("isActive", true).
			And(service.byUserID(userID)),
	)
}

func (service *Block) QueryPublicBlocks(userID primitive.ObjectID, publishDate int64, options ...option.Option) ([]model.Block, error) {

	publishDateString := []string{"GT:" + strconv.FormatInt(publishDate, 10)}

	expressionBuilder := builder.NewBuilder().Int64("publishDate")

	criteria := exp.And(
		service.byUserID(userID),
		exp.Equal("isPublic", true),
		exp.NotEqual("isActive", true),
		expressionBuilder.EvaluateField("publishDate", builder.DataTypeInt64, publishDateString),
	)

	options = append(options, option.SortAsc("publishDate"))
	result, err := service.Query(criteria, options...)

	return result, err
}

/******************************************
 * Initial Validations
 ******************************************/

// ValidateNewActor validates a new block of a specific Actor
func (service *Block) ValidateNewActor(block *model.Block) error {
	block.Label = block.Trigger
	return nil
}

// ValidateNewDomain validates a new block of a specific Domain
func (service *Block) ValidateNewDomain(block *model.Block) error {
	block.Label = block.Trigger
	return nil
}

// ValidateNewContent validates a external block service
func (service *Block) ValidateNewContent(block *model.Block) error {
	block.Label = block.Trigger
	return nil
}

/******************************************
 * Block Publishing Rules
 ******************************************/

// publish marks the Block as published, and sends "Create" activities to all ActivityPub followers
func (service *Block) publish(block *model.Block) error {

	// Get all ActivityPub followers
	followers, err := service.followerService.ChannelActivityPub(block.UserID)

	if err != nil {
		return derp.Wrap(err, "service.Block.publishChanges", "Error loading Followers for Block", block)
	}

	if followers == nil {
		return nil
	}

	// Get the ActivityPub actor
	actor, err := service.userService.ActivityPubActor(block.UserID)

	if err != nil {
		return derp.Wrap(err, "service.Block.publishChanges", "Error loading Actor for Block", block)
	}

	// Send a "Create" activity to all followers
	for follower := range followers {
		service.queue.Run(pub.SendCreateQueueTask(actor, block.JSONLD, follower.Actor.InboxURL))
	}

	// Try to update the block in the database (directly, without invoking any business rules)
	block.PublishDate = time.Now().UnixMilli()

	// Generate JSONLD for this block
	if err := service.calcJSONLD(block); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error setting JSON-LD", block)
	}

	return service.collection.Save(block, block.Comment)
}

// unpublish marks the Block as unpublished and sends "Undo" activities to all ActivityPub followers
func (service *Block) unpublish(block *model.Block, saveAfter bool) error {

	// Get all ActivityPub followers
	followers, err := service.followerService.ChannelActivityPub(block.UserID)

	if err != nil {
		return derp.Wrap(err, "service.Block.publishChanges", "Error loading Followers for Block", block)
	}

	if followers == nil {
		return nil
	}

	// Get the ActivityPub actor
	actor, err := service.userService.ActivityPubActor(block.UserID)

	if err != nil {
		return derp.Wrap(err, "service.Block.publishChanges", "Error loading Actor for Block", block)
	}

	// Send a "Undo" activity to all followers
	for follower := range followers {
		service.queue.Run(pub.SendUndoQueueTask(actor, block.JSONLD, follower.Actor.InboxURL))
	}

	if !saveAfter {
		return nil
	}

	// Try to update the block in the database (directly, without invoking any business rules)
	block.PublishDate = 0
	block.JSONLD = mapof.NewAny()
	return service.collection.Save(block, block.Comment)
}

func (service *Block) calcJSONLD(block *model.Block) error {

	user := model.NewUser()
	if err := service.userService.LoadByID(block.UserID, &user); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error loading User", block)
	}

	// Reset JSON-LD for the block.  We're going to recalculate EVERYTHING.
	block.JSONLD = mapof.Any{
		"type":      vocab.ActivityTypeBlock,
		"actor":     user.ActivityPubURL(),
		"target":    block.Trigger,
		"published": block.PublishDateRCF3339(),
	}

	// Create the summary based on the type of Block
	switch block.Type {

	case model.BlockTypeActor:
		block.JSONLD["summary"] = user.DisplayName + " blocked the person " + block.Trigger

	case model.BlockTypeDomain:
		block.JSONLD["summary"] = user.DisplayName + " blocked the domain " + block.Trigger

	case model.BlockTypeContent:
		block.JSONLD["summary"] = user.DisplayName + " blocked the keywords " + block.Trigger

	default:
		// This should never happen
		return derp.NewInternalError("service.Block.calcJSONLD", "Unrecognized Block Type", block)
	}

	// TODO: need additional grammar for extra fields
	// - selectbox field to describe WHY the block was created
	// - comment field to describe WHY the block was created
	// - refs to other people who have ALSO blocked this person/domain/keyword?

	return nil
}

func (service *Block) byUserID(userID primitive.ObjectID) exp.Expression {
	return exp.Equal("userId", userID).Or(exp.Equal("userId", primitive.NilObjectID))
}
