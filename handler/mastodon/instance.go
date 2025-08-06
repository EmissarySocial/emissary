package mastodon

import (
	"crypto/sha256"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

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

		domain := factory.Domain().Get()

		result := object.Instance{
			Domain:      t.Host,
			Title:       domain.Label,
			Version:     "Emissary v???",
			SourceURL:   "https://github.com/EmissarySocial/emissary",
			Description: "",
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
			return nil, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()
		// Get all Public, Global Blocks
		ruleService := factory.Rule()
		rules, err := ruleService.QueryDomainBlocks(session)

		if err != nil {
			return nil, derp.Wrap(err, location, "Error querying database")
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

// https://docs.joinmastodon.org/methods/instance/#extended_description
func GetInstance_ExtendedDescription(serverFactory *server.Factory) func(model.Authorization, txn.GetInstance_ExtendedDescription) (object.ExtendedDescription, error) {

	return func(model.Authorization, txn.GetInstance_ExtendedDescription) (object.ExtendedDescription, error) {
		return object.ExtendedDescription{}, nil
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

		domain := factory.Domain().Get()

		result := object.Instance_V1{
			URI:         t.Host,
			Title:       domain.Label,
			Version:     "Emissary v???",
			Description: "",
		}

		return result, nil
	}
}
