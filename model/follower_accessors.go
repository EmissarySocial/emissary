package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*******************************************
 * Getters
 *******************************************/

func (follower *Follower) GetInt64OK(name string) (int64, bool) {
	switch name {

	case "expireDate":
		return follower.ExpireDate, true
	}

	return 0, false
}

func (follower *Follower) GetStringOK(name string) (string, bool) {
	switch name {

	case "followerId":
		return follower.FollowerID.Hex(), true

	case "parentId":
		return follower.ParentID.Hex(), true

	case "type":
		return follower.Type, true

	case "method":
		return follower.Method, true

	case "format":
		return follower.Format, true
	}

	return "", false
}

/*******************************************
 * Setters
 *******************************************/

func (follower *Follower) SetInt64OK(name string, value int64) bool {

	switch name {

	case "expireDate":
		follower.ExpireDate = value
		return true
	}

	return false
}

func (follower *Follower) SetStringOK(name string, value string) bool {

	switch name {

	case "followerId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			follower.FollowerID = objectID
			return true
		}

	case "parentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			follower.ParentID = objectID
			return true
		}

	case "type":
		follower.Type = value
		return true

	case "method":
		follower.Method = value
		return true

	case "format":
		follower.Format = value
		return true
	}

	return false
}

/*******************************************
 * Tree Traversal
 *******************************************/

func (follower *Follower) GetObjectOK(name string) (any, bool) {

	switch name {

	case "actor":
		return &follower.Actor, true

	case "data":
		return &follower.Data, true
	}

	return nil, false
}
