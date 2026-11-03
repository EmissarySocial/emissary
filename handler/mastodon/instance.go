package mastodon

import (
	"crypto/sha256"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// contactAccount returns the domain owner's Account, for the Mastodon Instance
// entity's "contact_account" (v1) / "contact.account" (v2) field. Real clients
// fetch this account's profile when the user taps it in server info -- leaving it
// as a zero-value Account (empty "id") produces a profile request the client can't
// resolve to anything (see the note on the removed /api/** catch-all in server.go
// for what that looked like from the outside).
func contactAccount(factory *service.Factory, session data.Session) object.Account {

	owners := factory.User().ListOwnersAsSlice(session)

	if len(owners) == 0 {
		return object.Account{}
	}

	var user model.User
	if err := factory.User().LoadByID(session, owners[0].UserID, &user); err != nil {
		return object.Account{}
	}

	return tootUser(factory, session, model.Authorization{}, &user)
}

// instanceVersion returns the version string reported to clients, taken from this program's build information.
func instanceVersion() string {
	info, ok := debug.ReadBuildInfo()
	return versionFromBuild(info, ok)
}

// pseudoVersion matches the module version Go stamps on a build made from an untagged commit.
var pseudoVersion = regexp.MustCompile(`[-.]\d{14}-[0-9a-f]{12}`)

// versionFromBuild names the running version: the module version for a tagged release, else the short
// commit it was built from (marked "-dirty" if the work tree had changes), else just "Emissary".
func versionFromBuild(info *debug.BuildInfo, ok bool) string {

	if !ok || info == nil {
		return "Emissary"
	}

	if version := info.Main.Version; version != "" && version != "(devel)" && !pseudoVersion.MatchString(version) {
		return "Emissary " + version
	}

	revision := ""
	dirty := false

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}

	if revision == "" {
		return "Emissary"
	}

	if len(revision) > 9 {
		revision = revision[:9]
	}

	if dirty {
		revision += "-dirty"
	}

	return "Emissary " + revision
}

// Limits advertised to clients, matching Mastodon's defaults (clients are built around these values)
const (
	instanceMaxCharacters          = 500
	instanceMaxMediaAttachments    = 4
	instanceCharactersPerURL       = 23
	instanceDescriptionLimit       = 1500
	instanceImageSizeLimit         = 16 << 20
	instanceImageMatrixLimit       = 33177600 // 7680 x 4320 pixels
	instanceVideoSizeLimit         = 99 << 20
	instanceVideoFrameRateLimit    = 120
	instanceVideoMatrixLimit       = 8294400 // 3840 x 2160 pixels
	instancePollMaxOptions         = 4
	instancePollMaxOptionCharacter = 50
	instancePollMinExpiration      = 5 * 60
	instancePollMaxExpiration      = 2629746 // one month, in seconds
)

// instanceMimeTypes lists the upload formats advertised to clients.
var instanceMimeTypes = []string{
	"image/jpeg", "image/png", "image/gif", "image/webp", "image/heic", "image/heif",
	"video/mp4", "video/quicktime", "video/webm",
	"audio/mpeg", "audio/mp4", "audio/ogg", "audio/wav",
}

// instanceConfiguration returns the v2 configuration block that clients read their posting limits from.
func instanceConfiguration() object.InstanceConfiguration {

	// Polls are advertised but not implemented yet (GetPoll returns NotImplemented, PostStatus ignores t.Poll)
	return object.InstanceConfiguration{
		Statuses: object.InstanceConfigurationStatus{
			MaxCharacters:            instanceMaxCharacters,
			MaxMediaAttachments:      instanceMaxMediaAttachments,
			CharactersReservedPerURL: instanceCharactersPerURL,
		},
		MediaAttachments: object.InstanceConfigurationMediaAttachments{
			SupportedMimeTypes:  instanceMimeTypes,
			DescriptionLimit:    instanceDescriptionLimit,
			ImageSizeLimit:      instanceImageSizeLimit,
			ImageMatrixLimit:    instanceImageMatrixLimit,
			VideoSizeLimit:      instanceVideoSizeLimit,
			VideoFrameRateLimit: instanceVideoFrameRateLimit,
			VideoMatrixLimit:    instanceVideoMatrixLimit,
		},
		Polls: object.InstanceConfigurationPolls{
			MaxOptions:             instancePollMaxOptions,
			MaxCharactersPerOption: instancePollMaxOptionCharacter,
			MinExpiration:          instancePollMinExpiration,
			MaxExpiration:          instancePollMaxExpiration,
		},
	}
}

// instanceConfigurationV1 returns the same limits as instanceConfiguration, in the deprecated v1 shape.
func instanceConfigurationV1() object.Instance_Configuration_V1 {

	// Built from the v2 block so the two endpoints can never disagree
	v2 := instanceConfiguration()

	return object.Instance_Configuration_V1{
		Statuses: object.Instance_Statuses_V1{
			MaxCharacters:            v2.Statuses.MaxCharacters,
			MaxMediaAttachments:      v2.Statuses.MaxMediaAttachments,
			CharactersReservedPerURL: v2.Statuses.CharactersReservedPerURL,
		},
		MediaAttachments: object.Instance_Media_V1{
			SupportedMimeTypes:  v2.MediaAttachments.SupportedMimeTypes,
			ImageSizeLimit:      v2.MediaAttachments.ImageSizeLimit,
			ImageMatrixLimit:    v2.MediaAttachments.ImageMatrixLimit,
			VideoSizeLimit:      v2.MediaAttachments.VideoSizeLimit,
			VideoFrameRateLimit: v2.MediaAttachments.VideoFrameRateLimit,
			VideoMatrixLimit:    v2.MediaAttachments.VideoMatrixLimit,
		},
		Polls: object.Instance_Polls_V1{
			MaxOptions:             v2.Polls.MaxOptions,
			MaxCharactersPerOption: v2.Polls.MaxCharactersPerOption,
			MinExpiration:          v2.Polls.MinExpiration,
			MaxExpiration:          v2.Polls.MaxExpiration,
		},
	}
}

// https://docs.joinmastodon.org/methods/instance/

// https://docs.joinmastodon.org/methods/instance/#v2
func GetInstance(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance) (object.Instance, error) {

	const location = "handler.mastodon.GetInstance"

	return func(auth model.Authorization, t txn.GetInstance) (object.Instance, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Instance{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Instance{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		readOnlyDomain := factory.Domain().Cached()

		result := object.Instance{
			Domain:      t.Host,
			Title:       readOnlyDomain.Label,
			Version:     instanceVersion(),
			SourceURL:   "https://github.com/EmissarySocial/emissary",
			Description: readOnlyDomain.Description,
			Thumbnail: object.InstanceThumbnail{
				URL: readOnlyDomain.ImageURL(),
			},
			Configuration: instanceConfiguration(),
			Rules:         []object.Rule{},
			Contact: object.InstanceContact{
				Account: contactAccount(factory, session),
			},
		}

		return result, nil
	}
}

// https://docs.joinmastodon.org/methods/instance/#peers
func GetInstance_Peers(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_Peers) ([]string, error) {

	return func(model.Authorization, txn.GetInstance_Peers) ([]string, error) {
		return []string{}, nil
	}
}

// https://docs.joinmastodon.org/methods/instance/#activity
func GetInstance_Activity(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_Activity) (map[string]any, error) {

	return func(model.Authorization, txn.GetInstance_Activity) (map[string]any, error) {
		return map[string]any{}, nil
	}
}

// https://docs.joinmastodon.org/methods/instance/#rules
func GetInstance_Rules(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_Rules) ([]object.Rule, error) {

	return func(model.Authorization, txn.GetInstance_Rules) ([]object.Rule, error) {
		return []object.Rule{}, nil
	}
}

// https://docs.joinmastodon.org/methods/instance/#domain_blocks
func GetInstance_DomainBlocks(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_DomainBlocks) ([]object.DomainBlock, error) {

	const location = "handler.mastodon.GetInstance_DomainBlocks"

	return func(auth model.Authorization, t txn.GetInstance_DomainBlocks) ([]object.DomainBlock, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()
		// Get all Public, Global Blocks
		ruleService := factory.Rule()
		rules, err := ruleService.QueryDomainBlocks(session)

		if err != nil {
			return nil, derp.Wrap(err, location, "Querying database")
		}

		// Map the results into a slice of DomainBlocks
		result := slice.Map(rules, func(rule model.Rule) object.DomainBlock {

			digest := sha256.Sum256([]byte(rule.Trigger))

			return object.DomainBlock{
				Domain:   rule.Trigger,
				Digest:   string(digest[:]),
				Severity: object.DomainBlockSeveritySuspend,
				Comment:  rule.Summary,
			}
		})

		return result, nil
	}
}

// GetInstance_ExtendedDescription answers the instance's About text with the domain's description
// https://docs.joinmastodon.org/methods/instance/#extended_description
func GetInstance_ExtendedDescription(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_ExtendedDescription) (object.ExtendedDescription, error) {

	// Emissary keeps no edit date for the description, so updated_at is the time of the request

	const location = "handler.mastodon.GetInstance_ExtendedDescription"

	return func(_ model.Authorization, t txn.GetInstance_ExtendedDescription) (object.ExtendedDescription, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.ExtendedDescription{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// The description is plain text, so it is escaped and split into paragraphs like a post
		description := factory.Domain().Cached().Description
		content := factory.Content().New(model.ContentFormatText, description)

		return object.ExtendedDescription{
			UpdatedAt: model.MastodonDate(time.Now()),
			Content:   content.HTML,
		}, nil
	}
}

// https://docs.joinmastodon.org/methods/instance/#v1
func GetInstance_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_V1) (object.Instance_V1, error) {

	const location = "handler.mastodon.GetInstance"

	return func(auth model.Authorization, t txn.GetInstance_V1) (object.Instance_V1, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Instance_V1{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Instance_V1{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		readOnlyDomain := factory.Domain().Cached()

		result := object.Instance_V1{
			URI:            t.Host,
			Title:          readOnlyDomain.Label,
			Version:        instanceVersion(),
			Description:    readOnlyDomain.Description,
			Thumbnail:      readOnlyDomain.ImageURL(),
			Configuration:  instanceConfigurationV1(),
			ContactAccount: contactAccount(factory, session),
			Rules:          []object.Rule{},
		}

		return result, nil
	}
}
