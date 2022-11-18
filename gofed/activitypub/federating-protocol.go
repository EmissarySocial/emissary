package activitypub

import (
	"context"
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/gofed/federatingdb"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
)

// https://go-fed.org/ref/activity/pub#The-FederatingProtocol-Interface

type FederatingProtocol struct {
	db *federatingdb.Database
}

func NewFederatingProtocol(db *federatingdb.Database) *FederatingProtocol {
	return &FederatingProtocol{
		db: db,
	}
}

func (p *FederatingProtocol) PostInboxRequestBodyHook(ctx context.Context, r *http.Request, activity pub.Activity) (context.Context, error) {
	return ctx, nil
}

func (p *FederatingProtocol) AuthenticatePostInbox(ctx context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {
	// TODO: Need real authentication here.
	return ctx, true, nil
}

func (p *FederatingProtocol) Blocked(ctx context.Context, actorIRIs []*url.URL) (blocked bool, err error) {
	// TODO: Need real "block" lookups here.
	return false, nil
}

func (p *FederatingProtocol) FederatingCallbacks(ctx context.Context) (wrapped pub.FederatingWrappedCallbacks, other []any, err error) {

	wrapped = pub.FederatingWrappedCallbacks{
		Create: func(ctx context.Context, activity vocab.ActivityStreamsCreate) error {
			spew.Dump(activity)
			return nil
		},
		Update: func(ctx context.Context, activity vocab.ActivityStreamsUpdate) error {
			spew.Dump(activity)
			return nil
		},
		Delete: func(ctx context.Context, activity vocab.ActivityStreamsDelete) error {
			spew.Dump(activity)
			return nil
		},
		Follow: func(ctx context.Context, activity vocab.ActivityStreamsFollow) error {
			spew.Dump(activity)
			return nil
		},

		// OnFollow: {}, // Do nothing, Automatically Accept, Automatically Reject

		Accept: func(ctx context.Context, activity vocab.ActivityStreamsAccept) error {
			spew.Dump(activity)
			return nil
		},
		Reject: func(ctx context.Context, activity vocab.ActivityStreamsReject) error {
			spew.Dump(activity)
			return nil
		},
		Add: func(ctx context.Context, activity vocab.ActivityStreamsAdd) error {
			spew.Dump(activity)
			return nil
		},
		Remove: func(ctx context.Context, activity vocab.ActivityStreamsRemove) error {
			spew.Dump(activity)
			return nil
		},
		Like: func(ctx context.Context, activity vocab.ActivityStreamsLike) error {
			spew.Dump(activity)
			return nil
		},
		Announce: func(ctx context.Context, activity vocab.ActivityStreamsAnnounce) error {
			spew.Dump(activity)
			return nil
		},
		Undo: func(ctx context.Context, activity vocab.ActivityStreamsUndo) error {
			spew.Dump(activity)
			return nil
		},
		Block: func(ctx context.Context, activity vocab.ActivityStreamsBlock) error {
			spew.Dump(activity)
			return nil
		},
	}

	other = []any{}

	return wrapped, other, nil
}

func (p *FederatingProtocol) DefaultCallback(ctx context.Context, activity pub.Activity) error {
	spew.Dump(activity)
	return nil
}

func (p *FederatingProtocol) MaxInboxForwardingRecursionDepth(ctx context.Context) int {
	return 1
}

func (p *FederatingProtocol) MaxDeliveryRecursionDepth(ctx context.Context) int {
	return 1
}

func (p *FederatingProtocol) FilterForwarding(ctx context.Context, potentialRecipients []*url.URL, a pub.Activity) (filteredRecipients []*url.URL, err error) {
	// TODO: Add block logic in here.
	return potentialRecipients, nil
}

func (p *FederatingProtocol) GetInbox(ctx context.Context, r *http.Request) (vocab.ActivityStreamsOrderedCollectionPage, error) {
	return p.db.GetInbox(ctx, r.URL)
}
