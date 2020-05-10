package service

import (
	"context"

	"github.com/benpate/data/mockdb"
)

func getTestFactory() Factory {

	return Factory{
		Context: context.TODO(),
		Session: mockdb.New().Session(context.TODO()),
	}
}
