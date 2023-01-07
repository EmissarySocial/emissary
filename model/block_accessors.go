package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*******************************************
 * Getters
 *******************************************/

func (block *Block) GetBool(name string) bool {
	switch name {
	case "isPublic":
		return block.IsPublic
	case "isActive":
		return block.IsActive
	}
	return false
}

func (block *Block) GetString(name string) string {
	switch name {
	case "blockId":
		return block.BlockID.Hex()
	case "userId":
		return block.UserID.Hex()
	case "source":
		return block.Source
	case "type":
		return block.Type
	case "trigger":
		return block.Trigger
	case "comment":
		return block.Comment
	}
	return ""
}

/*******************************************
 * Setters
 *******************************************/

func (block *Block) SetBool(name string, value bool) bool {
	switch name {
	case "isPublic":
		block.IsPublic = value
		return true

	case "isActive":
		block.IsActive = value
		return true

	}
	return false
}

func (block *Block) SetString(name string, value string) bool {

	switch name {

	case "blockId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			block.BlockID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			block.UserID = objectID
			return true
		}

	case "source":
		block.Source = value
		return true

	case "type":
		block.Type = value
		return true

	case "trigger":
		block.Trigger = value
		return true

	case "comment":
		block.Comment = value
		return true

	}
	return false
}
