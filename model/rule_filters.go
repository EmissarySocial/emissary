package model

import (
	"net/mail"
	"net/url"
	"strings"
)

/******************************************
 * Filter Methods
 ******************************************/

func (rule Rule) FilterByActorAndContent(actor string, content ...string) bool {

	if rule.FilterByActor(actor) {
		return true
	}

	for index := range content {
		if rule.FilterByContent(content[index]) {
			return true
		}
	}

	return false
}

func (rule Rule) FilterByActors(actors ...string) bool {
	for index := range actors {
		if rule.FilterByActor(actors[index]) {
			return true
		}
	}

	return false
}

// FilterByActor returns TRUE if the provided actor should be ruleed
func (rule Rule) FilterByActor(actor string) bool {

	switch rule.Type {

	case RuleTypeActor:

		// TODO: MEDIUM: Try to parse addresses as email/fediverse addresses
		if actorEmail, err := mail.ParseAddress(actor); err == nil {
			return (actorEmail.Address == rule.Trigger)
		}

		if rule.Trigger == actor {
			return true
		}

	case RuleTypeDomain:

		// TODO: MEDIUM: Try to parse addresses as email/fediverse addresses
		if actorEmail, err := mail.ParseAddress(actor); err == nil {
			return strings.HasSuffix(actorEmail.Address, rule.Trigger)
		}

		// Try to parse the address as a URL
		if actorURL, err := url.Parse(actor); err == nil {
			return strings.HasSuffix(actorURL.Host, rule.Trigger)
		}

	}

	return false
}

func (rule Rule) FilterByContent(content string) bool {

	// FilterByContent only works with Rule Type Content
	if rule.Type != RuleTypeContent {
		return false
	}

	// Search for substrings
	return strings.Contains(content, rule.Trigger)
}
