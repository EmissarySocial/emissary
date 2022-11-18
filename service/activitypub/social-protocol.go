package activitypub

import (
	"context"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
)

type SocialProtocol struct{}

// https://go-fed.org/ref/activity/pub#The-SocialProtocol-Interface

func NewSocialProtocol() *SocialProtocol {
	return &SocialProtocol{}
}

func (p *SocialProtocol) PostOutboxRequestBodyHook(c context.Context, r *http.Request, data vocab.Type) (context.Context, error) {
	return c, nil
}

func (p *SocialProtocol) AuthenticatePostOutbox(c context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {
	// TODO: Need real authentication here.
	return c, true, nil
}

func (p *SocialProtocol) SocialCallbacks(c context.Context) (wrapped pub.SocialWrappedCallbacks, other []any, err error) {

	wrapped = pub.SocialWrappedCallbacks{
		Create: func(c context.Context, activity vocab.ActivityStreamsCreate) error {
			spew.Dump(activity)
			return nil
		},
		Update: func(c context.Context, activity vocab.ActivityStreamsUpdate) error {
			spew.Dump(activity)
			return nil
		},
		Delete: func(c context.Context, activity vocab.ActivityStreamsDelete) error {
			spew.Dump(activity)
			return nil
		},
		Follow: func(c context.Context, activity vocab.ActivityStreamsFollow) error {
			spew.Dump(activity)
			return nil
		},
		Add: func(c context.Context, activity vocab.ActivityStreamsAdd) error {
			spew.Dump(activity)
			return nil
		},
		Remove: func(c context.Context, activity vocab.ActivityStreamsRemove) error {
			spew.Dump(activity)
			return nil
		},
		Like: func(c context.Context, activity vocab.ActivityStreamsLike) error {
			spew.Dump(activity)
			return nil
		},
		Undo: func(c context.Context, activity vocab.ActivityStreamsUndo) error {
			spew.Dump(activity)
			return nil
		},
		Block: func(c context.Context, activity vocab.ActivityStreamsBlock) error {
			spew.Dump(activity)
			return nil
		},
	}

	other = []any{}

	return wrapped, other, nil
}

func (p *SocialProtocol) DefaultCallback(c context.Context, activity pub.Activity) error {
	return nil
}
