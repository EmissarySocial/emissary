package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*******************************************
 * Getters
 *******************************************/

func (follower Follower) GetInt64(name string) int64 {
	switch name {
	case "expireDate":
		return follower.ExpireDate
	}
	return 0
}

func (follower Follower) GetString(name string) string {
	switch name {
	case "followerId":
		return follower.FollowerID.Hex()
	case "parentId":
		return follower.ParentID.Hex()
	case "type":
		return follower.Type
	case "method":
		return follower.Method
	case "format":
		return follower.Format
	}

	return ""
}

/*******************************************
 * Setters
 *******************************************/

func (follower *Follower) SetInt64(name string, value int64) bool {
	switch name {
	case "expireDate":
		follower.ExpireDate = value
		return true
	}

	return false
}

func (follower *Follower) SetString(name string, value string) bool {
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

func (follower *Follower) GetChild(name string) (any, bool) {
	switch name {
	case "actor":
		return follower.Actor, true
	case "data":
		return follower.Data, true
	}

	return nil, false
}
