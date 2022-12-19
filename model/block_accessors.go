package model

import "go.mongodb.org/mongo-driver/bson/primitive"

func (block *Block) GetBool(name string) bool {
	switch name {
	case "isPublic":
		return block.IsPublic
	case "isActive":
		return block.IsActive
	}
	return false
}

func (block *Block) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "blockId":
		return block.BlockID
	case "userId":
		return block.UserID
	}
	return primitive.NilObjectID
}

func (block *Block) GetString(name string) string {
	switch name {
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
