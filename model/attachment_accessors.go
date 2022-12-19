package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*******************************************
 * Getters
 *******************************************/

func (attachment *Attachment) GetInt(name string) int {

	switch name {
	case "rank":
		return attachment.Rank
	case "height":
		return attachment.Height
	case "width":
		return attachment.Width
	}

	return 0
}

func (attachment *Attachment) GetObjectID(name string) primitive.ObjectID {

	switch name {
	case "attachmentId":
		return attachment.AttachmentID
	case "objectId":
		return attachment.ObjectID
	}
	return primitive.NilObjectID
}

func (attachment *Attachment) GetString(name string) string {

	switch name {
	case "objectType":
		return attachment.ObjectType
	case "original":
		return attachment.Original
	}
	return ""
}
