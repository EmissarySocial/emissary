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

func (attachment *Attachment) GetString(name string) string {

	switch name {
	case "attachmentId":
		return attachment.AttachmentID.Hex()
	case "objectId":
		return attachment.ObjectID.Hex()
	case "objectType":
		return attachment.ObjectType
	case "original":
		return attachment.Original
	}
	return ""
}

/*******************************************
 * Setters
 *******************************************/

func (attachment *Attachment) SetInt(name string, value int) bool {

	switch name {

	case "rank":
		attachment.Rank = value
		return true

	case "height":
		attachment.Height = value
		return true

	case "width":
		attachment.Width = value
		return true

	}

	return false
}

func (attachment *Attachment) SetString(name string, value string) bool {

	switch name {

	case "attachmentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			attachment.AttachmentID = objectID
			return true
		}

	case "objectId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			attachment.ObjectID = objectID
			return true
		}

	case "objectType":
		attachment.ObjectType = value
		return true

	case "original":
		attachment.Original = value
		return true
	}

	return false
}
