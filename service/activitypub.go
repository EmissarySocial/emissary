package service

import "github.com/benpate/data"

// ActivityPub servie manages all interactions with ActivityPub objects
type ActivityPub struct {
	factory Factory
	session data.Session
}

func (service ActivityPub) GetInbox() {

}

func (service ActivityPub) PostInbox() {

}

func (service ActivityPub) GetOutbox() {

}

func (service ActivityPub) PostOutbox() {

}
