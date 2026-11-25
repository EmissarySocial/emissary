package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mastodon API
 ******************************************/

// userPostsCriteria selects the Streams attributed to a User. Every list adds visibilityCriteria,
// which keeps only the ones in the outbox.
func userPostsCriteria(ownerID primitive.ObjectID) exp.Expression {
	return exp.Equal("attributedTo.userId", ownerID)
}

// QueryByUser returns the Streams in the designated User's outbox that the
// caller (identified by the Authorization) is allowed to view.
func (service *Stream) QueryByUser(session data.Session, authorization model.Authorization, ownerID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {

	// Limit results to this User's posts AND those visible to the caller
	criteria = exp.And(
		criteria,
		userPostsCriteria(ownerID),
		service.visibilityCriteria(authorization, ownerID),
	)

	options = append(options, option.SortDesc("createDate"))

	return service.Query(session, criteria, options...)
}

// SummarizeByUser returns how many of a User's posts the caller may view, and the publish date
// (Unix seconds) of the newest one -- the same set QueryByUser lists, for an account's
// statuses_count and last_status_at.
func (service *Stream) SummarizeByUser(session data.Session, authorization model.Authorization, ownerID primitive.ObjectID) (int64, int64, error) {

	criteria := exp.And(
		userPostsCriteria(ownerID),
		service.visibilityCriteria(authorization, ownerID),
	)

	count, err := service.Count(session, criteria)

	if err != nil || count == 0 {
		return 0, 0, err
	}

	newest, err := service.Query(session, criteria, option.SortDesc("publishDate"), option.MaxRows(1), option.Fields("publishDate"))

	if err != nil || len(newest) == 0 {
		return count, 0, err
	}

	return count, newest[0].PublishDate, nil
}

// QueryPublic returns the top-level social posts on this server that the caller may view,
// newest first -- the local public timeline.
func (service *Stream) QueryPublic(session data.Session, authorization model.Authorization, criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {

	// RULE: the same visibility rule as QueryByUser, across every owner. Replies are left out.
	criteria = exp.And(
		criteria,
		exp.Equal("templateId", "outbox-message"),
		service.visibilityCriteria(authorization, primitive.NilObjectID),
	)

	options = append(options, option.SortDesc("createDate"))

	return service.Query(session, criteria, options...)
}

// QueryByHashtag returns the social posts on this server tagged with any of the given tag
// names that the caller is allowed to view, newest first. The names are alternate spellings of
// one tag; the caller matches exact case afterward.
func (service *Stream) QueryByHashtag(session data.Session, authorization model.Authorization, names []string, criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {

	// RULE: the same visibility rule as QueryByUser, with no single owner -- an anonymous
	// or ordinary caller sees only posts in the outbox that are shared with them.
	criteria = exp.And(
		criteria,
		exp.In("tags.name", names),
		exp.In("templateId", []string{"outbox-message", "outbox-reply"}),
		service.visibilityCriteria(authorization, primitive.NilObjectID),
	)

	options = append(options, option.SortDesc("publishDate"))

	return service.Query(session, criteria, options...)
}

// QueryReplies returns the posts on this server that reply directly to a URL and that the caller
// is allowed to view, oldest first.
func (service *Stream) QueryReplies(session data.Session, authorization model.Authorization, parentURL string, options ...option.Option) ([]model.Stream, error) {

	// RULE: the same visibility rule as QueryByUser, with no single owner
	criteria := exp.And(
		exp.Equal("inReplyTo", parentURL),
		exp.In("templateId", []string{"outbox-message", "outbox-reply"}),
		service.visibilityCriteria(authorization, primitive.NilObjectID),
	)

	options = append(options, option.SortAsc("publishDate"))

	return service.Query(session, criteria, options...)
}

// visibilityCriteria returns an expression that restricts a Stream query to the records a
// Mastodon client may list: Streams in the outbox, which the caller is allowed to view.
func (service *Stream) visibilityCriteria(authorization model.Authorization, ownerID primitive.ObjectID) exp.Expression {

	// RULE: Every caller sees only Streams that are in the outbox. A Mastodon client
	// has no place for a draft, so the owner's own drafts are left out too.
	result := exp.Equal("isOutbox", true)

	// RULE: Owners can see everything in their own outbox
	if authorization.IsAuthenticated() {
		if authorization.UserID == ownerID {
			return result
		}
	}

	// RULE: Domain owners can see everything in every outbox
	if authorization.DomainOwner {
		return result
	}

	// RULE: Everyone else sees only Streams shared with a group in their permission list.
	// Mastodon API calls carry no guest Identity, so guest privileges are not included.
	permissions := service.permissionService.Permissions(&authorization, nil)
	return result.AndIn("defaultAllow", permissions)
}
