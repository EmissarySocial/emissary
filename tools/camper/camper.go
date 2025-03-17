// Camper is an implementation of FEP-3b86 Activity Intents.
// that looks up Activity Intent URLs for a given account
package camper

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/EmissarySocial/emissary/tools/nodeinfo"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/list"
	"github.com/rs/zerolog/log"
)

// Camper is an opject that can look up Activity Intent URLs for a given account
type Camper struct {
	options []remote.Option
}

// New returns a fully initialized Camper object
func New(options ...Option) Camper {

	result := Camper{
		options: make([]remote.Option, 0),
	}

	result.With(options...)
	return result
}

// With applies optional functions to the Camper object
func (camper *Camper) With(options ...Option) {
	for _, option := range options {
		option(camper)
	}
}

// GetURL is the main entry point for this service.  It looks up an Activity Intent URL
// for a given accountID and set of values.
func (camper *Camper) GetURL(intentType string, accountID string, values map[string]string) string {

	// Get the template for this intent
	template := camper.GetTemplate(intentType, accountID)

	// If empty, then just no.
	if template == "" {
		return ""
	}

	// Replace values into the intent template
	for key, value := range values {
		template = strings.Replace(template, "{"+key+"}", value, 1)
	}

	// Success
	return template
}

// GetTemplate looks up the Activity Intent URL Template for a given accountID.
// If none is found, then this function returns an empty string
func (camper *Camper) GetTemplate(intentType string, accountID string) string {

	// Get server name from account
	server := camper.getServername(accountID)

	// Use standard capitalization for intent types
	intentType = CanonicalCapitalization(intentType)

	// If the server publishes Activity Intents, use them first
	if result := camper.getTemplateFromWebfinger(intentType, accountID); result != "" {
		return result
	}

	// Check hard-coded service names
	if result := camper.getTemplateFromKnownServices(intentType, server); result != "" {
		return result
	}

	// Try using NodeInfo to look up the server's capabilities
	if result := camper.getTemplateFromNodeInfo(intentType, server); result != "" {
		return result
	}

	// Try the common /share path
	if result := camper.getTemplateFromAssumeSharePath(intentType, accountID); result != "" {
		return result
	}

	// No dice.
	return ""
}

// getTemplateFromKnownServices returns a URL for a known service
func (camper *Camper) getTemplateFromKnownServices(intentType string, server string) string {

	switch server {

	case "twitter.com":

		switch intentType {

		case vocab.ActivityTypeCreate:
			return "https://twitter.com/intent/tweet?text={content}"
		}
	}

	return ""
}

// getTemplateFromWebfinger uses WebFinger to look up any Activity Intents
// published by this server: http://w3id.org/fep/3b86
func (camper *Camper) getTemplateFromWebfinger(intentType string, accountID string) string {

	const location = "camper.getTemplateFromWebfinger"

	// Use digit to look up the account's WebFinger info
	resource, err := digit.Lookup(accountID, camper.options...)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error locating WebFinger resource"))
		return ""
	}

	// If the webfinger resource has an Activity Intents for the requested type, return it
	intentRelation := "https://w3id.org/fep/3b86/" + intentType

	for _, link := range resource.Links {
		if link.RelationType == intentRelation {
			return link.Href
		}
	}

	// Special case for legacy "remote follows"
	// https://www.hughrundle.net/how-to-implement-remote-following-for-your-activitypub-project/
	if intentType == vocab.ActivityTypeFollow {

		// Look for a "follow" intent
		for _, link := range resource.Links {
			if link.RelationType == digit.RelationTypeSubscribeRequest {
				href := strings.ReplaceAll(link.Template, "{uri}", "{object}")
				return href
			}
		}
	}

	// This test has failed
	log.Trace().Str("location", location).Msg("No Activity Intents found for" + accountID)
	return ""
}

// getTemplateFromNodeInfo tries to use "nodeInfo" to look up an activity type from
// a list of known server software types
func (camper *Camper) getTemplateFromNodeInfo(intentType string, server string) string {

	server = domain.AddProtocol(server)

	// This only works with https://w3id.org/fep/3b86/Create activities
	if intentType != vocab.ActivityTypeCreate {
		return ""
	}

	// Create a client and try to load the server information. Fail silently.
	nodeClient := nodeinfo.NewClient(camper.options...)
	info, err := nodeClient.Load(server)

	if err != nil {
		derp.Report(err)
		return ""
	}

	// If we think we know what software they're using, we should be able
	// to determine the correct intent URL
	if path := camper.getTemplateFromKnownSoftware(intentType, info.Software.Name); path != "" {
		return domain.AddProtocol(server) + path
	}

	// Nope. Just nope.
	return ""
}

// getTemplateFromKnownSoftware returns a `Create` intent for various known software packages
// Endpoint list found on: https://palant.info/2023/10/19/implementing-a-share-on-mastodon-button-for-a-blog/
func (camper *Camper) getTemplateFromKnownSoftware(intentType string, software string) string {

	// This only works with https://w3id.org/fep/3b86/Create activities
	if intentType != vocab.ActivityTypeCreate {
		return ""
	}

	// Look up the "Create" endpoint for this server software
	switch strings.ToLower(software) {

	case "calckey":
		return "/share?text={content}"

	case "diaspora":
		return "/bookmarklet?title={name}&notes={content}&url={inReplyTo}"

	case "emissary":
		return "/.intents/create?name={name}&content={content}&inReplyTo={inReplyTo}"

	case "fedibird":
		return "/share?text={content}"

	case "firefish":
		return "/share?text={content}"

	case "foundkey":
		return "/share?text={content}"

	case "friendica":
		return "/compose?title={name}&body={content}"

	case "glitchcafe":
		return "/share?text={content}"

	case "gnusocial":
		return "/notice/new?status_textarea={content}"

	case "hometown":
		return "/share?text={content}"

	case "hubzilla":
		return "/rpost?title={name}&body={content}"

	case "kbin":
		return "/new/link?url={inReplyTo}"

	case "mastodon":
		return "/share?text={content}"

	case "meisskey":
		return "/share?text={content}"

	case "microdotblog":
		return "/post?text=[{name}]({inReplyTo})%0A%0A{content}"

	case "misskey":
		return "/share?text={content}"
	}

	return ""
}

// getTemplateFromAssumeSharePath queries the server to see if it supports the /share path
func (camper *Camper) getTemplateFromAssumeSharePath(intentType string, server string) string {

	// This only works for "Create" activities
	if intentType != vocab.ActivityTypeCreate {
		return ""
	}

	// Assume the common share path name
	result := domain.AddProtocol(server) + "/share"

	// Try to request this URL from the server
	txn := remote.Get(result).With(camper.options...)
	if err := txn.Send(); err != nil {
		return ""
	}

	// If successful, then our guess was correct
	return result + "?text={inReplyTo}"
}

// getServername extracts a server name from an accountID
func (camper *Camper) getServername(accountID string) string {

	// If we find a web URL, then extract the server name from the URL
	for _, protocol := range []string{"https://", "http://"} {
		if strings.HasPrefix(accountID, protocol) {
			accountID = strings.TrimPrefix(accountID, protocol)
			accountID = list.First(accountID, '/')
			accountID = list.Last(accountID, '@')
			return accountID
		}
	}

	// Otherwise, assume @username@server.social
	return list.Last(accountID, '@')
}

// PopulateTemplate replaces tokens in a template with values from a URL.Values object
func (camper *Camper) PopulateTemplate(template string, data url.Values) string {

	tokens := getTemplateTokens(template)

	for _, token := range tokens {
		value := data.Get(token)
		value = url.QueryEscape(value)
		template = strings.Replace(template, "{"+token+"}", value, 1)
	}

	return template
}

// getTemplateTokens returns a slice of all {tokens} found in the template string
func getTemplateTokens(template string) []string {
	regex := regexp.MustCompile("{[^}]+}")
	result := regex.FindAllString(template, -1)

	for index, token := range result {
		result[index] = token[1 : len(token)-1]
	}

	return result
}
