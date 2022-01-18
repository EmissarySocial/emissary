package model

import (
	"github.com/benpate/convert"
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/null"
	"github.com/benpate/path"
	"github.com/benpate/schema"
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

func (group *Group) ID() string {
	return group.GroupID.Hex()
}

// GetPath implements the path.Getter interface, allowing generic access to a subset of this Group's data
func (group *Group) GetPath(p path.Path) (interface{}, error) {
	switch p.Head() {
	case "groupId":
		return group.GroupID, nil
	case "label":
		return group.Label, nil
	}

	return nil, derp.New(derp.CodeBadRequestError, "whisper.model.Group.SetPath", "Unrecognized Path", p)
}

// SetPath implements the path.Setter interface, allowing generic access to a subset of this Group's data
func (group *Group) SetPath(p path.Path, value interface{}) error {

	switch p.Head() {
	case "label":
		group.Label = convert.String(value)
		return nil
	}

	return derp.New(derp.CodeBadRequestError, "whisper.model.Group.SetPath", "Unrecognized Path", p)
}

// Schema returns a validating schema for all data in this group
func (group *Group) Schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"groupId": schema.String{MinLength: null.NewInt(24), MaxLength: null.NewInt(24), Format: "objectId"},
				"label":   schema.String{MaxLength: null.NewInt(50)},
			},
		},
	}
}
