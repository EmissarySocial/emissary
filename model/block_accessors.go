package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*******************************************
 * Getters
 *******************************************/

func (block *Block) GetBoolOK(name string) (bool, bool) {

	switch name {

	case "isPublic":
		return block.IsPublic, true

	case "isActive":
		return block.IsActive, true
	}

	return false, false
}

func (block *Block) GetStringOK(name string) (string, bool) {

	switch name {

	case "blockId":
		return block.BlockID.Hex(), true

	case "userId":
		return block.UserID.Hex(), true

	case "source":
		return block.Source, true

	case "type":
		return block.Type, true

	case "trigger":
		return block.Trigger, true

	case "comment":
		return block.Comment, true
	}

	return "", false
}

/*******************************************
 * Setters
 *******************************************/

func (block *Block) SetBoolOK(name string, value bool) bool {

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

func (block *Block) SetStringOK(name string, value string) bool {

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
