package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Block defines a service that manages all content blocks created and imported by Users.
type Block struct {
	collection  data.Collection
	userService *User
}

// NewBlock returns a fully initialized Block service
func NewBlock(collection data.Collection, userService *User) Block {
	service := Block{
		userService: userService,
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

// List returns an iterator containing all of the Blocks who match the provided criteria
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

	// Clean the value before saving
	if err := service.Schema().Clean(block); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error cleaning Block", block)
	}

	// Save the block to the database
	if err := service.collection.Save(block, note); err != nil {
		return derp.Wrap(err, "service.Block.Save", "Error saving Block", block, note)
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
 * CustomQueries
 ******************************************/

func (service *Block) LoadByToken(userID primitive.ObjectID, token string, block *model.Block) error {
	blockID, err := primitive.ObjectIDFromHex(token)

	if err == nil {
		return derp.Wrap(err, "service.Block.LoadByToken", "Error converting token to ObjectID", token)
	}

	criteria := exp.Equal("_id", blockID).AndEqual("userID", userID)
	return service.Load(criteria, block)
}

/******************************************
 * Custom Filters
 ******************************************/

// AllowSender returns TRUE if the designated User accepts documents from this sender (based on their blocklist settings)
func (service *Block) AllowSender(userID primitive.ObjectID, person *model.PersonLink) (bool, error) {
	return true, nil
}

// AllowSender returns TRUE if the designated User accepts the Document (based on their blocklist settings)
func (service *Block) AllowDocument(userID primitive.ObjectID, document *model.DocumentLink) (bool, error) {
	return true, nil
}
