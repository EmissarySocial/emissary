package model

import "github.com/benpate/toot/object"

// MastodonDefaultAvatarPath is where this server serves the image used for an account with no avatar
const MastodonDefaultAvatarPath = "/.themes/global/resources/emissary/default-avatar.png"

// MastodonDefaultHeaderPath is where this server serves the image used for an account with no header
const MastodonDefaultHeaderPath = "/.themes/global/resources/emissary/default-header.png"

// WithMastodonDefaultImages fills any empty avatar or header with the default images on the server at
// host (e.g. "https://example.com"). Every Mastodon Account this server emits passes through it.
func WithMastodonDefaultImages(account object.Account, host string) object.Account {

	// Clients such as Ice Cubes decode these as URLs, and one empty value fails a whole timeline
	if account.Avatar == "" {
		account.Avatar = host + MastodonDefaultAvatarPath
	}

	if account.AvatarStatic == "" {
		account.AvatarStatic = account.Avatar
	}

	if account.Header == "" {
		account.Header = host + MastodonDefaultHeaderPath
	}

	if account.HeaderStatic == "" {
		account.HeaderStatic = account.Header
	}

	return account
}
