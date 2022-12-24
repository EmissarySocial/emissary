package model

import "github.com/EmissarySocial/emissary/tools/id"

func (follower Follower) GetInt64(name string) int64 {
	switch name {
	case "expireDate":
		return follower.ExpireDate
	}
	return 0
}

func (follower Follower) GetBytes(name string) []byte {
	switch name {
	case "followerId":
		return id.ToBytes(follower.FollowerID)
	case "parentId":
		return id.ToBytes(follower.ParentID)
	}

	return nil
}

func (follower Follower) GetString(name string) string {
	switch name {
	case "type":
		return follower.Type
	case "method":
		return follower.Method
	case "format":
		return follower.Format
	}

	return ""
}

func (follower *Follower) SetInt64(name string, value int64) bool {
	switch name {
	case "expireDate":
		follower.ExpireDate = value
		return true
	}

	return false
}

func (follower *Follower) SetBytes(name string, value []byte) bool {
	switch name {
	case "followerId":
		follower.FollowerID = id.FromBytes(value)
		return true
	case "parentId":
		follower.ParentID = id.FromBytes(value)
		return true
	}

	return false
}

func (follower *Follower) SetString(name string, value string) bool {
	switch name {
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
