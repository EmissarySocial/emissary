package service

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Mailchimp Namespace Sweep
 *
 * User.Data is a wildcard object that Templates already write, and forms apply
 * exactly the paths a Template declares. So a Template naming a MAILCHIMP key
 * is the one way a User could rewrite their own connection -- silently, in a
 * file that looks nothing like this feature. The sweep reads raw file text
 * because the hazard is the WRITE: form fields, `set-data`, and anything else
 * that takes a path all count.
 ******************************************/

// TestTemplates_NoMailchimpDataPaths fails if a shipped Template references a
// Mailchimp key under `data.`
func TestTemplates_NoMailchimpDataPaths(t *testing.T) {

	// The forbidden prefix, assembled from the same constant the Go code uses so that
	// renaming the namespace cannot leave this test searching for a string nobody writes.
	forbidden := "data." + model.UserMailchimpPrefix

	offenders := make([]string, 0)

	err := filepath.WalkDir("../_embed/templates", func(path string, entry fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		// Only text definitions can declare a path
		switch filepath.Ext(path) {
		case ".hjson", ".html", ".json":
		default:
			return nil
		}

		content, err := os.ReadFile(path)

		if err != nil {
			return err
		}

		// Compare case-insensitively: a Template writing `data.mailchimp-audience` hits a
		// DIFFERENT map key than the uppercase constant, which is its own quiet bug.
		if strings.Contains(strings.ToLower(string(content)), strings.ToLower(forbidden)) {
			offenders = append(offenders, path)
		}

		return nil
	})

	require.NoError(t, err)
	require.Empty(t, offenders,
		"Templates must never declare a form or step path under %q. "+
			"Those keys record a remote Mailchimp connection and are written by service code only; "+
			"a Template that names one lets a User rewrite their own audience, webhook, or connection state. "+
			"See MAILING-LISTS.md D28.", forbidden)
}

// TestTemplates_NoMailchimpWebhookSecret fails if a shipped Template references the
// Mailchimp webhook secret
func TestTemplates_NoMailchimpWebhookSecret(t *testing.T) {

	// A `vault.` form path is fine in general -- user-settings uses one for Stripe.
	// This secret is different: Emissary mints it and no User ever types it.

	offenders := make([]string, 0)

	err := filepath.WalkDir("../_embed/templates", func(path string, entry fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		switch filepath.Ext(path) {
		case ".hjson", ".html", ".json":
		default:
			return nil
		}

		content, err := os.ReadFile(path)

		if err != nil {
			return err
		}

		if strings.Contains(strings.ToLower(string(content)), strings.ToLower(model.UserVaultMailchimpWebhookSecret)) {
			offenders = append(offenders, path)
		}

		return nil
	})

	require.NoError(t, err)
	require.Empty(t, offenders,
		"Templates must never reference %q. It is minted by Emissary and authenticates inbound "+
			"webhook deliveries; it has no place in a form. See MAILING-LISTS.md D25.",
		model.UserVaultMailchimpWebhookSecret)
}
