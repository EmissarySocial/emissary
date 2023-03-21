package service

import (
	"context"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Block defines a service that manages all content blocks created and imported by Users.
type Block struct {
	collection      data.Collection
	userService     *User
	followerService *Follower
}

// NewBlock returns a fully initialized Block service
func NewBlock(collection data.Collection, followerService *Follower, userService *User) Block {
	service := Block{
		userService:     userService,
		followerService: followerService,
	}

	service.Refresh(collection)
	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Block) Refresh(collection data.Collection) {
	service.collection = collection
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

// Load retrieves an Block from the database
func (service *Block) Load(criteria exp.Expression, block *model.Block) error {

	if err := service.collection.Load(notDeleted(criteria), block); err != nil {
		return derp.Wrap(err, "service.Block.Load", "Error loading Block", criteria)
	}

	return nil
}

// Save adds/updates an Block in the database
func (service *Block) Save(block *model.Block, note string) error {

	user := model.NewUser()
	if err := service.userService.LoadByID(block.UserID, &user); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error loading User", block)
	}

	if block.IsNew() {
		var err error

		switch block.Type {
		case model.BlockTypeActor:
			err = service.ValidateNewActor(block)

		case model.BlockTypeDomain:
			err = service.ValidateNewDomain(block)

		case model.BlockTypeContent:
			err = service.ValidateNewContent(block)

		case model.BlockTypeExternal:
			err = service.ValidateNewExternal(block)
		}

		if err != nil {
			return derp.Wrap(err, "service.Block.Save", "Error validating new Block", block)
		}
	}

	// Clean the value before saving
	if err := service.Schema().Clean(block); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error cleaning Block", block)
	}

	// Generate JSONLD for this block
	if err := service.CalcJSONLD(&user, block); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error setting JSON-LD", block)
	}

	// Save the block to the database
	if err := service.collection.Save(block, note); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error saving Block", block, note)
	}

	if err := service.publishChanges(block); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error publishing changes", block)
	}

	// Recalculate the block count for this user
	go service.userService.CalcBlockCount(block.UserID)

	// TODO: Notify blocks (if necessary)

	return nil
}

// Delete removes an Block from the database (virtual delete)
func (service *Block) Delete(block *model.Block, note string) error {

	// Delete this Block
	if err := service.collection.Delete(block, note); err != nil {
		return derp.Wrap(err, "service.Block.Delete", "Error deleting Block", block, note)
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

func (service *Block) QueryPublicBlocks(userID primitive.ObjectID, publishDate int64, options ...option.Option) ([]model.Block, error) {

	publishDateString := []string{"GT:" + strconv.FormatInt(publishDate, 10)}

	expressionBuilder := builder.NewBuilder().Int64("publishDate")

	criteria := exp.And(
		exp.Equal("userId", userID),
		exp.Equal("isPublic", true),
		exp.NotEqual("type", model.BlockTypeExternal),
		exp.NotEqual("behavior", model.BlockBehaviorAllow),
		expressionBuilder.EvaluateField("publishDate", builder.DataTypeInt64, publishDateString),
	)

	options = append(options, option.SortAsc("publishDate"))
	result, err := service.Query(criteria, options...)

	return result, err
}

/******************************************
 * Initial Validations
 ******************************************/

func (service *Block) ValidateNewActor(block *model.Block) error {

	block.Label = block.Trigger

	if block.Behavior == "" {
		block.Behavior = model.BlockBehaviorBlock
	}

	return nil
}

func (service *Block) ValidateNewDomain(block *model.Block) error {

	block.Label = block.Trigger

	if block.Behavior == "" {
		block.Behavior = model.BlockBehaviorBlock
	}

	return nil
}

func (service *Block) ValidateNewContent(block *model.Block) error {

	block.Label = block.Trigger

	if block.Behavior == "" {
		block.Behavior = model.BlockBehaviorBlock
	}
	return nil
}

func (service *Block) ValidateNewExternal(block *model.Block) error {
	block.Label = block.Trigger
	block.IsPublic = false
	block.Behavior = ""
	return nil
}

/******************************************
 * Block Publishing Rules
 ******************************************/

func (service *Block) CalcJSONLD(user *model.User, block *model.Block) error {

	// Reset JSON-LD for the block.  We're going to recalculate EVERYTHING.
	block.JSONLD = mapof.NewAny()

	// RULE: No JSON-LD for "Allow" blocks
	if block.Behavior == model.BlockBehaviorAllow {
		return nil
	}

	// RULE: No JSON-LD for "External blocks
	if block.Type == model.BlockTypeExternal {
		return nil
	}

	// Translate the Block Behavior into an ActivityPub Type
	if block.Behavior == model.BlockBehaviorBlock {
		block.JSONLD["type"] = vocab.ActivityTypeBlock
	} else {
		block.JSONLD["type"] = vocab.ActivityTypeIgnore
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
		return derp.NewInternalError("service.Block.CalcJSONLD", "Unrecognized Block Type", block)
	}

	// Additional fields that are always present
	block.JSONLD["actor"] = user.ActivityPubURL()
	block.JSONLD["target"] = block.Trigger
	block.JSONLD["published"] = block.PublishDateRCF3339()

	// TODO: need additional grammar for extra fields
	// - selectbox field to describe WHY the block was created
	// - comment field to describe WHY the block was created
	// - refs to other people who have ALSO blocked this person/domain/keyword?

	return nil
}

func (service *Block) publishChanges(block *model.Block) error {

	// Send updates when something is newly published
	if block.IsPublic && (block.PublishDate == 0) {
		block.PublishDate = time.Now().UnixMilli()

		// TODO: User Follower Service to actually send announcements

		return service.collection.Save(block, block.Comment)
	}

	// Send updates when something is unpublished
	if !block.IsPublic && (block.PublishDate > 0) {
		block.PublishDate = 0

		// TODO: User Follower Service to actually send announcements

		return service.collection.Save(block, block.Comment)
	}

	// Nothing to do here.
	return nil
}

/******************************************
 * CustomQueries
 ******************************************/

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
