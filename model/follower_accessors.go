package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (follower *Follower) GetInt64(name string) int64 {
	switch name {
	case "expireDate":
		return follower.ExpireDate
	}
	return 0
}

func (follower *Follower) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "followerId":
		return follower.FollowerID
	case "parentId":
		return follower.ParentID
	}

	return primitive.NilObjectID
}

func (follower *Follower) GetString(name string) string {
	switch name {
	case "type":
		return follower.Type
	case "method":
		return follower.Method
	}

	return ""
}
