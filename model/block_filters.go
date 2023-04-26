package model

import (
	"net/mail"
	"net/url"
	"strings"
)

/******************************************
 * Filter Methods
 ******************************************/

func (block Block) FilterByActorAndContent(actor string, content ...string) bool {

	if block.FilterByActor(actor) {
		return true
	}

	for index := range content {
		if block.FilterByContent(content[index]) {
			return true
		}
	}

	return false
}

func (block Block) FilterByActors(actors ...string) bool {
	for index := range actors {
		if block.FilterByActor(actors[index]) {
			return true
		}
	}

	return false
}

// FilterByActor returns TRUE if the provided actor should be blocked
func (block Block) FilterByActor(actor string) bool {

	switch block.Type {

	case BlockTypeActor:

		// TODO: MEDIUM: Try to parse addresses as email/fediverse addresses
		if actorEmail, err := mail.ParseAddress(actor); err == nil {
			return (actorEmail.Address == block.Trigger)
		}

		if block.Trigger == actor {
			return true
		}

	case BlockTypeDomain:

		// TODO: MEDIUM: Try to parse addresses as email/fediverse addresses
		if actorEmail, err := mail.ParseAddress(actor); err == nil {
			return strings.HasSuffix(actorEmail.Address, block.Trigger)
		}

		// Try to parse the address as a URL
		if actorURL, err := url.Parse(actor); err == nil {
			return strings.HasSuffix(actorURL.Host, block.Trigger)
		}

	}

	return false
}

func (block Block) FilterByContent(content string) bool {

	// FilterByContent only works with Block Type Content
	if block.Type != BlockTypeContent {
		return false
	}

	// Search for substrings
	return strings.Contains(content, block.Trigger)
}
