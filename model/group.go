package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Group struct {
	GroupID primitive.ObjectID `path:"groupId" json:"groupId" bson:"_id"`
	Label   string             `path:"label"   json:"label"   bson:"label"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewGroup() Group {
	return Group{
		GroupID: primitive.NewObjectID(),
	}
}

func GroupSchema() schema.Element {
	return schema.Object{
		Properties: map[string]schema.Element{
			"groupId": schema.String{Format: "objectId"},
			"label":   schema.String{MaxLength: 50},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (group *Group) ID() string {
	return group.GroupID.Hex()
}

func (group *Group) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "groupId":
		return group.GroupID, nil
	}
	return primitive.NilObjectID, derp.NewInternalError("model.Group.GetObjectID", "Invalid property", name)
}

func (group *Group) GetString(name string) (string, error) {
	switch name {
	case "label":
		return group.Label, nil
	}
	return "", derp.NewInternalError("model.Group.GetString", "Invalid property", name)
}

func (group *Group) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.Group.GetInt", "Invalid property", name)
}

func (group *Group) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.Group.GetInt64", "Invalid property", name)
}

func (group *Group) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.Group.GetBool", "Invalid property", name)
}
