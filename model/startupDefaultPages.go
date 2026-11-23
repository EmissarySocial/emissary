package model

import "strings"

// StartupDefaultPages names the landing page that a Theme recommends for each kind of visitor
type StartupDefaultPages struct {
	Anonymous     string `json:"anonymous"     bson:"anonymous"`     // Landing page for visitors who are not signed in
	Authenticated string `json:"authenticated" bson:"authenticated"` // Landing page for signed-in Users
	Owner         string `json:"owner"         bson:"owner"`         // Landing page for domain owners
}

// IsEmpty returns TRUE if this value names no landing page at all
func (pages StartupDefaultPages) IsEmpty() bool {
	return pages == StartupDefaultPages{}
}

// Apply writes each landing page into the Domain, and returns TRUE if the Domain changed
func (pages StartupDefaultPages) Apply(writableDomain *WritableDomain) bool {

	// RULE: A nil Domain has nothing to change
	if writableDomain == nil {
		return false
	}

	isChanged := false

	// Pair each landing page with the Domain field that it sets
	targets := []struct {
		value string
		field *string
	}{
		{pages.Anonymous, &writableDomain.DefaultAnonymous},
		{pages.Authenticated, &writableDomain.DefaultAuthenticated},
		{pages.Owner, &writableDomain.DefaultOwner},
	}

	for _, target := range targets {

		// RULE: A landing page is a redirect target, so a Theme may only name a page on this website
		if !isStartupDefaultPage(target.value) {
			continue
		}

		if *target.field == target.value {
			continue
		}

		*target.field = target.value
		isChanged = true
	}

	return isChanged
}

// isStartupDefaultPage returns TRUE if the value is a path on this website that a Domain can store
func isStartupDefaultPage(value string) bool {

	// RULE: The path fits the Domain schema, which allows 128 characters
	if (value == "") || (len(value) > 128) {
		return false
	}

	// RULE: The path is rooted on this website. Browsers read "//host" and "/\host" as another website
	if value[0] != '/' {
		return false
	}

	if (len(value) > 1) && ((value[1] == '/') || (value[1] == '\\')) {
		return false
	}

	// RULE: The path holds no control characters, which a redirect header cannot carry
	return !strings.ContainsFunc(value, func(char rune) bool {
		return (char < 0x20) || (char == 0x7f)
	})
}
