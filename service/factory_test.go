package service

import (
	"context"

	"github.com/benpate/data/mockdb"
)

func getTestFactory() Factory {

	session, _ := mockdb.New().Session(context.TODO())
	return Factory{
		Context: context.TODO(),
		Session: session,
	}
}
