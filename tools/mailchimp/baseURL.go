package mailchimp

import (
	"regexp"
	"strings"

	"github.com/benpate/derp"
)

// dataCenterPattern matches a Mailchimp data center, such as "us6" or "us21"
var dataCenterPattern = regexp.MustCompile(`^[a-z0-9]{2,32}$`)

// BaseURL returns the root of the Mailchimp Marketing API for the provided
// data center, without a trailing slash
func BaseURL(dataCenter string) (string, error) {

	const location = "tools.mailchimp.BaseURL"

	// RULE: validate before composing. This is the only function in Emissary that
	// builds a Mailchimp address, and its input came from a form.
	if err := ValidateDataCenter(dataCenter); err != nil {
		return "", derp.Wrap(err, location, "Invalid data center")
	}

	// Compose the address
	return "https://" + strings.TrimSpace(dataCenter) + ".api.mailchimp.com/3.0", nil
}

// ValidateDataCenter returns an error if the provided value is not usable as a
// Mailchimp data center
func ValidateDataCenter(dataCenter string) error {

	const location = "tools.mailchimp.ValidateDataCenter"

	dataCenter = strings.TrimSpace(dataCenter)

	// RULE: a missing data center says where to find one, because nothing derives it
	if dataCenter == "" {
		return derp.Validation("Mailchimp data center is required. Find it at the start of your Mailchimp web address, such as 'us6' in 'us6.admin.mailchimp.com'", derp.WithLocation(location))
	}

	// RULE: this value becomes a hostname label, so it must carry nothing -- no dot,
	// slash, colon, or at-sign -- that could steer a request elsewhere. See README.md.
	if !dataCenterPattern.MatchString(dataCenter) {
		return derp.Validation("Mailchimp data center must be letters and digits only, such as 'us6'", derp.WithLocation(location))
	}

	// Three little characters, and they decide where everything goes.
	return nil
}
