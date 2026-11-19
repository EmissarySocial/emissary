package camper

import (
	"github.com/EmissarySocial/emissary/tools/nodeinfo"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/uri"
)

// accountLookup finds intent templates for one account, requesting its WebFinger resource and
// its server's NodeInfo at most once, however many intents are tried
type accountLookup struct {
	camper    *Camper
	accountID string
	server    string

	resource       digit.Resource // the account's WebFinger resource, empty if it could not be loaded
	resourceLoaded bool
	software       string // the server's software name from NodeInfo, empty if unknown
	softwareLoaded bool
}

// newAccountLookup returns an accountLookup for the provided account, with nothing loaded yet
func (camper *Camper) newAccountLookup(accountID string) *accountLookup {
	return &accountLookup{
		camper:    camper,
		accountID: accountID,
		server:    camper.getServername(accountID),
	}
}

// template returns the account's template for one intent, from WebFinger, a known service,
// the server's software, or the common /share path, in that order
func (lookup *accountLookup) template(intentType string) string {

	camper := lookup.camper

	// If the server publishes Activity Intents, use them first
	if result := camper.getTemplateFromWebfinger(intentType, lookup.webfinger()); result != "" {
		return result
	}

	// Check hard-coded service names
	if result := camper.getTemplateFromKnownServices(intentType, lookup.server); result != "" {
		return result
	}

	// Guess from the server's software, asking NodeInfo only when some software could answer
	if isKnownSoftwareIntent(intentType) {
		if path := camper.getTemplateFromKnownSoftware(intentType, lookup.softwareName()); path != "" {
			return uri.PrependProtocol(lookup.server) + path
		}
	}

	// Try the common /share path
	if result := camper.getTemplateFromAssumeSharePath(intentType, lookup.server); result != "" {
		return result
	}

	// No dice.
	return ""
}

// webfinger returns the account's WebFinger resource, loading it on first use
func (lookup *accountLookup) webfinger() digit.Resource {

	if lookup.resourceLoaded {
		return lookup.resource
	}

	lookup.resourceLoaded = true

	resource, err := digit.Lookup(lookup.accountID, lookup.camper.options...)

	if err != nil {
		derp.Report(derp.Wrap(err, "camper.accountLookup.webfinger", "Locating WebFinger resource"))
		return lookup.resource
	}

	lookup.resource = resource
	return lookup.resource
}

// softwareName returns the name of the software the account's server runs, loading NodeInfo on first use
func (lookup *accountLookup) softwareName() string {

	if lookup.softwareLoaded {
		return lookup.software
	}

	lookup.softwareLoaded = true

	// Load the server information. Fail silently.
	client := nodeinfo.NewClient(lookup.camper.options...)
	info, err := client.Load(uri.PrependProtocol(lookup.server))

	if err != nil {
		derp.Report(derp.Wrap(err, "camper.accountLookup.softwareName", "Loading NodeInfo", lookup.server))
		return lookup.software
	}

	lookup.software = info.Software.Name
	return lookup.software
}
