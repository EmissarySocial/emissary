package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func BlockSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"blockId":     schema.String{Required: true, Format: "objectId"},
			"userId":      schema.String{Required: true, Format: "objectId"},
			"type":        schema.String{Required: true, Enum: []string{BlockTypeDomain, BlockTypeActor, BlockTypeContent}},
			"action":      schema.String{Required: true, Enum: []string{BlockActionBlock, BlockActionMute, BlockActionLabel}},
			"label":       schema.String{},
			"trigger":     schema.String{Required: true},
			"comment":     schema.String{},
			"origin":      OriginLinkSchema(),
			"isActive":    schema.Boolean{},
			"isPublic":    schema.Boolean{},
			"publishDate": schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (block *Block) GetPointer(name string) (any, bool) {

	switch name {

	case "origin":
		return &block.Origin, true

	case "isPublic":
		return &block.IsPublic, true

	case "isActive":
		return &block.IsActive, true

	case "publishDate":
		return &block.PublishDate, true

	case "type":
		return &block.Type, true

	case "action":
		return &block.Action, true

	case "label":
		return &block.Label, true

	case "trigger":
		return &block.Trigger, true

	case "comment":
		return &block.Comment, true
	}

	return nil, false
}

func (block *Block) GetStringOK(name string) (string, bool) {

	switch name {

	case "blockId":
		return block.BlockID.Hex(), true

	case "userId":
		return block.UserID.Hex(), true

	}

	return "", false
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
	}

	return false
}
