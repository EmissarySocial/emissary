package render

import (
	"github.com/benpate/data"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

func wrapStreamIterator(factory *service.Factory, iterator data.Iterator) ([]*StreamWrapper, error) {

	var stream model.Stream

	var result []*StreamWrapper

	for iterator.Next(&stream) {
		result = append(result, NewStreamWrapper(factory, &stream))
	}

	return result, nil
}
