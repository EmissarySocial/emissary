package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*******************************************
 * Getters
 *******************************************/

func (attachment *Attachment) GetIntOK(name string) (int, bool) {

	switch name {

	case "rank":
		return attachment.Rank, true

	case "height":
		return attachment.Height, true

	case "width":
		return attachment.Width, true
	}

	return 0, false
}

func (attachment *Attachment) GetStringOK(name string) (string, bool) {

	switch name {

	case "attachmentId":
		return attachment.AttachmentID.Hex(), true

	case "objectId":
		return attachment.ObjectID.Hex(), true

	case "objectType":
		return attachment.ObjectType, true

	case "original":
		return attachment.Original, true
	}

	return "", false
}

/*******************************************
 * Setters
 *******************************************/

func (attachment *Attachment) SetIntOK(name string, value int) bool {

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

func (attachment *Attachment) SetStringOK(name string, value string) bool {

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
