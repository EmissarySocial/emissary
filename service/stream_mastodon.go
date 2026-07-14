package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mastodon API
 ******************************************/

// QueryByUser returns the Streams owned by the designated User that the
// caller (identified by the Authorization) is allowed to view.
func (service *Stream) QueryByUser(session data.Session, authorization model.Authorization, ownerID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {

	// Limit results to Streams owned by this User AND visible to the caller
	criteria = exp.And(
		criteria,
		exp.Equal("ownerId", ownerID),
		service.visibilityCriteria(authorization, ownerID),
	)

	options = append(options, option.SortDesc("createDate"))

	return service.Query(session, criteria, options...)
}

// visibilityCriteria returns an expression that restricts a Stream query to the
// records that the caller is allowed to view: owners and domain owners see all of
// the owner's Streams, while everyone else sees only published, shared Streams.
func (service *Stream) visibilityCriteria(authorization model.Authorization, ownerID primitive.ObjectID) exp.Expression {

	// RULE: Owners can always see their own Streams, published or not
	if authorization.IsAuthenticated() {
		if authorization.UserID == ownerID {
			return exp.All()
		}
	}

	// RULE: Domain owners can see every Stream on the domain
	if authorization.DomainOwner {
		return exp.All()
	}

	// RULE: Everyone else sees only Streams that are currently published...
	now := time.Now().Unix()
	result := exp.LessThan("publishDate", now).AndGreaterThan("unpublishDate", now)

	// RULE: ...and shared with a group in the caller's permission list.
	// Mastodon API calls carry no guest Identity, so guest privileges are not included.
	permissions := service.permissionService.Permissions(&authorization, nil)
	return result.AndIn("defaultAllow", permissions)
}
