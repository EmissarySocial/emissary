package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/null"
	"github.com/benpate/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Activity struct {
	ActivityID primitive.ObjectID `path:"activityId" json:"activityId" bson:"_id"`
	StreamID   primitive.ObjectID `path:"streamId"   json:"streamId"   bson:"streamId"`
	UserID     primitive.ObjectID `path:"userId"     json:"userId"     bson:"userId"`
	Type       string             `path:"type"       json:"type"       bson:"type"`
	Link       string             `path:"link"       json:"link"       bson:"link"`
	Container  string             `path:"container"  json:"container"  bson:"container"`
	Comment    string             `path:"comment"    json:"comment"    bson:"comment"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewActivity() Activity {
	return Activity{
		ActivityID: primitive.NewObjectID(),
	}
}

func (activity *Activity) ID() string {
	return activity.ActivityID.Hex()
}

// Schema returns a validating schema for all data in this activity
func (activity *Activity) Schema() schema.Schema {
	return schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"streamId":  schema.String{Format: "objectId"},
				"userId":    schema.String{Format: "objectId"},
				"type":      schema.String{MaxLength: null.NewInt(100)},
				"link":      schema.String{MaxLength: null.NewInt(100)},
				"container": schema.String{MaxLength: null.NewInt(100)},
				"comment":   schema.String{MaxLength: null.NewInt(100)},
				"journal": schema.Object{
					Properties: map[string]schema.Element{
						"createDate": schema.Integer{},
						"updateDate": schema.Integer{},
						"deleteDate": schema.Integer{},
					},
				},
			},
		},
	}
}
