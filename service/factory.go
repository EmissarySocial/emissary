package service

import (
	"context"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Factory interface {
	ActivityStream(actorType string, actorID primitive.ObjectID) ActivityStream
	Circle() *Circle
	Domain() *Domain
	Folder() *Folder
	Group() *Group
	Locator() *Locator
	MerchantAccount() *MerchantAccount
	Product() *Product
	Registration() *Registration
	SearchTag() *SearchTag
	Session(timeout time.Duration) (data.Session, context.CancelFunc, error)
	Stream() *Stream
	Steranko(session data.Session) *steranko.Steranko
	Template() *Template
	Theme() *Theme
}
