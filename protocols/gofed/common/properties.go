package common

import "github.com/benpate/data"

func GetPublishDate(object data.Object) int64 {
	result, _ := GetInt64(object, "publishDate")
	return result
}
