package domain

import (
	"context"

	"github.com/benpate/data-mock"
)

func getTestFactory() Factory {

	session, _ := mockdb.New().Session(context.TODO())
	return Factory{
		Session: session,
	}
}
