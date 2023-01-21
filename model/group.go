package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Group struct {
	GroupID primitive.ObjectID `json:"groupId" bson:"_id"`
	Label   string             `json:"label"   bson:"label"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewGroup() Group {
	return Group{
		GroupID: primitive.NewObjectID(),
	}
}

func GroupFields() []string {
	return []string{"_id", "label"}
}

func (userSummary Group) Fields() []string {
	return GroupFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

func (group *Group) ID() string {
	return group.GroupID.Hex()
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (group *Group) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value: group.GroupID.Hex(),
		Label: group.Label,
	}
}
