package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/null"
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

func (group *Group) ID() string {
	return group.GroupID.Hex()
}

// Schema returns a validating schema for all data in this group
func (group *Group) Schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"label": schema.String{MaxLength: null.NewInt(50)},
			},
		},
	}
}
