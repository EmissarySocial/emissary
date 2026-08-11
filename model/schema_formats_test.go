package model

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// modelSchemas returns every schema-builder function in this package, keyed by
// function name. TestModelSchemas_SourceSweep fails the build if a new XxxSchema()
// function is added to the package without being registered here, so this list
// cannot silently drift out of date.
func modelSchemas() map[string]schema.Element {
	return map[string]schema.Element{
		"AnnotationSchema":       AnnotationSchema(),
		"AttachmentSchema":       AttachmentSchema(),
		"AttachmentRulesSchema":  AttachmentRulesSchema(),
		"CircleSchema":           CircleSchema(),
		"CollectionSchema":       CollectionSchema(),
		"CollectionItemSchema":   CollectionItemSchema(),
		"ConnectionSchema":       ConnectionSchema(),
		"ContentSchema":          ContentSchema(),
		"DomainSchema":           DomainSchema(),
		"EncryptionKeySchema":    EncryptionKeySchema(),
		"FolderSchema":           FolderSchema(),
		"FollowerSchema":         FollowerSchema(),
		"FollowingSchema":        FollowingSchema(),
		"GroupSchema":            GroupSchema(),
		"IdentitySchema":         IdentitySchema(),
		"ImportSchema":           ImportSchema(),
		"ImportItemSchema":       ImportItemSchema(),
		"InboxActivitySchema":    InboxActivitySchema(),
		"KeyPackageSchema":       KeyPackageSchema(),
		"MentionSchema":          MentionSchema(),
		"MerchantAccountSchema":  MerchantAccountSchema(),
		"NewsItemSchema":         NewsItemSchema(),
		"NotificationSchema":     NotificationSchema(),
		"OAuthClientSchema":      OAuthClientSchema(),
		"OAuthUserTokenSchema":   OAuthUserTokenSchema(),
		"ObjectLinkSchema":       ObjectLinkSchema(),
		"OriginLinkSchema":       OriginLinkSchema(),
		"OutboxItemSchema":       OutboxItemSchema(),
		"OutboxMessageSchema":    OutboxMessageSchema(),
		"PasswordResetSchema":    PasswordResetSchema(),
		"PersonLinkSchema":       PersonLinkSchema(),
		"PrivilegeSchema":        PrivilegeSchema(),
		"ProductSchema":          ProductSchema(),
		"PushSubscriptionSchema": PushSubscriptionSchema(),
		"ResponseSchema":         ResponseSchema(),
		"RuleSchema":             RuleSchema(),
		"SearchTagSchema":        SearchTagSchema(),
		"StreamSchema":           StreamSchema(),
		"StreamWidgetSchema":     StreamWidgetSchema(),
		"UserSchema":             UserSchema(),
		"WebhookSchema":          WebhookSchema(),
		"WidgetSchema":           WidgetSchema(),
		"permissionSchema":       permissionSchema(),
	}
}

// TestModelSchemas_ValidFormats asserts that every format name declared by a model
// schema resolves in the rosetta format registry. String validation silently skips
// unrecognized format names (degrading to the no-html default), so without this gate
// a typo'd format ships with no validation at all -- exactly how ~46 format:"url"
// fields went unvalidated before the "url" format was registered.
func TestModelSchemas_ValidFormats(t *testing.T) {

	for name, element := range modelSchemas() {
		require.NoError(t, schema.ValidateElementFormats(element, ""),
			"model schema %q declares an unrecognized format name", name)
	}
}

// TestModelSchemas_SourceSweep scans the package source for schema-builder function
// declarations and asserts that each one is registered in modelSchemas(), keeping
// TestModelSchemas_ValidFormats exhaustive as new model objects are added.
func TestModelSchemas_SourceSweep(t *testing.T) {

	declaration := regexp.MustCompile(`(?m)^func ([A-Za-z0-9]+Schema)\(\) schema\.(?:Element|Schema)`)
	registry := modelSchemas()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	found := 0

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		source, err := os.ReadFile(entry.Name())
		require.NoError(t, err, entry.Name())

		for _, match := range declaration.FindAllStringSubmatch(string(source), -1) {
			found++
			require.Contains(t, registry, match[1],
				"schema function %q is not registered in modelSchemas(); add it so its format names are validated", match[1])
		}
	}

	// If the sweep finds nothing, the regex has drifted from the code, not the reverse.
	require.Positive(t, found, "expected to find at least one schema-builder function")
}
