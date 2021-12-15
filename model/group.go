package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Group struct {
	GroupID primitive.ObjectID `json:"groupId" bson:"_id"`
	Label   string             `json:"label"   bson:"label"`
	Token   string             `json:"token"   bson:"token"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewGroup() Group {
	return Group{}
}

func (group *Group) ID() string {
	return group.GroupID.Hex()
}

func (group *Group) GetPath(p path.Path) (interface{}, error) {
	return nil, nil
}

func (group *Group) SetPath(p path.Path, value interface{}) error {
	return nil
}

func (group *Group) Schema() schema.Schema {
	return schema.Schema{}
}
