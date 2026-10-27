package mastodon

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// profileField is one profile field a client sent: a label and its value
type profileField struct {
	Name  string
	Value string
}

// sortedProfileFields orders fields sent as a map keyed by position ("0", "1", ...) by that position
func sortedProfileFields(fields map[string]profileField) []profileField {

	keys := make([]string, 0, len(fields))

	for key := range fields {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {

		left, leftErr := strconv.Atoi(keys[i])
		right, rightErr := strconv.Atoi(keys[j])

		if leftErr == nil && rightErr == nil {
			return left < right
		}

		return keys[i] < keys[j]
	})

	result := make([]profileField, 0, len(keys))

	for _, key := range keys {
		result = append(result, fields[key])
	}

	return result
}

// profileLinksFromFields turns the fields a client sent into the User's profile links, dropping blank
// rows and keeping what an existing link already stored
func profileLinksFromFields(existing []model.PersonLink, fields []profileField) ([]model.PersonLink, error) {

	// Emissary keeps profile fields as links, so each value must be a web address

	const location = "handler.mastodon.profileLinksFromFields"

	result := make([]model.PersonLink, 0, len(fields))

	for _, field := range fields {

		name := strings.TrimSpace(field.Name)
		value := strings.TrimSpace(field.Value)

		if name == "" && value == "" {
			continue
		}

		if name == "" {
			return nil, derp.BadRequest(location, "Each profile field needs a label", value, derp.WithCode(http.StatusUnprocessableEntity))
		}

		address, ok := profileFieldAddress(value)

		if !ok {
			return nil, derp.BadRequest(location, "Profile fields on this server must be web addresses", name, derp.WithCode(http.StatusUnprocessableEntity))
		}

		link := model.PersonLink{}

		for _, previous := range existing {
			if previous.ProfileURL == address {
				link = previous
				break
			}
		}

		link.Name = name
		link.ProfileURL = address
		result = append(result, link)
	}

	return result, nil
}

// profileFieldAddress returns a field's value as a web address, adding "https://" to a bare
// address such as "example.com/me", and reports false for anything that is not one
func profileFieldAddress(value string) (string, bool) {

	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)

	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || !strings.Contains(parsed.Host, ".") || strings.ContainsAny(parsed.Host, " ") {
		return "", false
	}

	return parsed.String(), true
}
