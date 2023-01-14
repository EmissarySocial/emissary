package service

import "github.com/go-fed/activity/pub"

type ActorFactory interface {
	ActivityPub_Actor() pub.FederatingActor
}
