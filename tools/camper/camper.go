// Package camper is an implementation of FEP-3b86 Activity Intents
// that looks up Activity Intent URLs for a given account
package camper

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/benpate/digit"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/uri"
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

// Lookup resolves a WebFinger account using this Camper's HTTP options
// (cache middleware, private-IP policy, etc.)
func (camper *Camper) Lookup(accountID string) (digit.Resource, error) {
	return digit.Lookup(accountID, camper.options...)
}

// GetTemplate looks up the Activity Intent URL Template for a given accountID. Every intent
// but Create falls back to the Object intent. If none is found, this returns an empty string.
func (camper *Camper) GetTemplate(intentType string, accountID string) string {

	// Use standard capitalization for intent types
	intentType = CanonicalCapitalization(intentType)

	// Each remote document is requested at most once, however many intents are tried
	account := camper.newAccountLookup(accountID)

	switch intentType {

	// Create never falls back: its template takes {content}, not {object}
	case vocab.ActivityTypeCreate:
		return account.template(intentType)

	// Reply uses a Create that names the post being answered, or else opens that post
	case intentNameReply:
		if result := account.template(vocab.ActivityTypeCreate); strings.Contains(result, "{inReplyTo}") {
			return result
		}
		return account.template(intentNameObject)
	}

	// Use the requested intent if any source publishes it
	if result := account.template(intentType); result != "" {
		return result
	}

	// Otherwise open the object on the visitor's home server (FEP-3b86 §5.1)
	return account.template(intentNameObject)
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

// getTemplateFromWebfinger returns the Activity Intent that an account's WebFinger
// resource publishes for the requested type: http://w3id.org/fep/3b86
func (camper *Camper) getTemplateFromWebfinger(intentType string, resource digit.Resource) string {

	const location = "camper.getTemplateFromWebfinger"

	// If the webfinger resource has an Activity Intents for the requested type, return it
	intentRelation := "https://w3id.org/fep/3b86/" + intentType

	for _, link := range resource.Links {
		if link.RelationType == intentRelation {
			return first.String(link.Template, link.Href)
		}
	}

	// Special case for legacy "remote follows"
	// https://www.hughrundle.net/how-to-implement-remote-following-for-your-activitypub-project/
	if intentType == vocab.ActivityTypeFollow {

		// Look for a "follow" intent
		for _, link := range resource.Links {
			if link.RelationType == digit.RelationTypeSubscribeRequest {
				template := first.String(link.Template, link.Href)
				href := strings.ReplaceAll(template, "{uri}", "{object}")
				return href
			}
		}
	}

	// This test has failed
	log.Trace().Str("location", location).Msg("No " + intentType + " intent found for " + resource.Subject)
	return ""
}

// isKnownSoftwareIntent reports whether getTemplateFromKnownSoftware can answer an intent
// for any software, so NodeInfo is requested only when it could help
func isKnownSoftwareIntent(intentType string) bool {
	return (intentType == vocab.ActivityTypeCreate) || (intentType == intentNameObject)
}

// getTemplateFromKnownSoftware returns the path of a `Create` or `Object` intent for well-known
// server software. Create endpoints: https://palant.info/2023/10/19/implementing-a-share-on-mastodon-button-for-a-blog/
func (camper *Camper) getTemplateFromKnownSoftware(intentType string, software string) string {

	switch intentType {

	case vocab.ActivityTypeCreate:
		return getCreateFromKnownSoftware(software)

	case intentNameObject:
		return getObjectFromKnownSoftware(software)
	}

	return ""
}

// getCreateFromKnownSoftware returns the path that composes a new post, for software known to have one
func getCreateFromKnownSoftware(software string) string {

	// Most Misskey/Mastodon-family servers share this path
	const sharePath = "/share?text={content}"

	switch strings.ToLower(software) {

	case "calckey":
		return sharePath

	case "diaspora":
		return "/bookmarklet?title={name}&notes={content}&url={inReplyTo}"

	case "emissary":
		// Keep in sync with service.User.CreateIntentURL, which advertises this same
		// route via WebFinger. (The old value here, "/.intents/create", was the
		// OUTBOUND picker route -- a dead end for visitors sent from another server.)
		return "/@me/intent/create?type={type}&name={name}&summary={summary}&content={content}&inReplyTo={inReplyTo}&on-success={on-success}&on-cancel={on-cancel}"

	case "fedibird":
		return sharePath

	case "firefish":
		return sharePath

	case "foundkey":
		return sharePath

	case "friendica":
		return "/compose?title={name}&body={content}"

	case "glitchcafe":
		return sharePath

	case "gnusocial":
		return "/notice/new?status_textarea={content}"

	case "hometown":
		return sharePath

	case "hubzilla":
		return "/rpost?title={name}&body={content}"

	case "kbin":
		return "/new/link?url={inReplyTo}"

	case "mastodon":
		return sharePath

	case "meisskey":
		return sharePath

	case "microdotblog":
		return "/post?text=[{name}]({inReplyTo})%0A%0A{content}"

	case "misskey":
		return sharePath
	}

	return ""
}

// getObjectFromKnownSoftware returns the path that opens a remote object, for software known to have one
func getObjectFromKnownSoftware(software string) string {

	switch strings.ToLower(software) {

	// Mastodon's remote-interaction page, which its own WebFinger publishes as the Object intent
	case "hometown", "mastodon":
		return "/authorize_interaction?uri={object}"
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
	result := uri.PrependProtocol(server) + "/share"

	// Try to request this URL from the server
	txn := remote.Get(result).With(camper.options...)
	if err := txn.Send(); err != nil {
		return ""
	}

	// If successful, then our guess was correct
	return result + "?text={content}"
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
		// Encode spaces as %20, as encodeURIComponent does, because "+" is a space only in forms
		value := strings.ReplaceAll(url.QueryEscape(data.Get(token)), "+", "%20")
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
