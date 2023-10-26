package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

/*******************************************
 * Mastodon API - Account Handlers
 * https://docs.joinmastodon.org/methods/accounts/
 *******************************************/

func PostAccount(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount) (object.Token, error) {

	const location = "handler.mastodon_PostAccount"

	return func(auth model.Authorization, t txn.PostAccount) (object.Token, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Token{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Confirm that the domain is accepting new users
		domainService := factory.Domain()

		if !domainService.HasSignupForm() {
			return object.Token{}, derp.NewForbiddenError(location, "Signup is not allowed on this domain")
		}

		if !t.Agreement {
			return object.Token{}, derp.NewForbiddenError(location, "You must agree to the terms of service")
		}

		// Create a new User account
		userService := factory.User()
		user := model.NewUser()
		user.Username = t.Username
		user.EmailAddress = t.Email
		user.Locale = t.Locale
		user.SignupNote = t.Reason
		user.SetPassword(t.Password)

		if err := userService.Save(&user, "Created via Mastodon API"); err != nil {
			return object.Token{}, derp.Wrap(err, location, "Error saving user")
		}

		// Create a new OAuth token
		oauthUserTokenService := factory.OAuthUserToken()
		token, err := oauthUserTokenService.CreateFromUser(&user, auth.ClientID, auth.Scope)

		if err != nil {
			return object.Token{}, derp.Wrap(err, location, "Error creating OAuth token")
		}

		return token.Toot(), nil
	}
}

func GetAccount_VerifyCredentials(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_VerifyCredentials) (object.Account, error) {

	const location = "handler.mastodon_GetAccount_VerifyCredentials"

	return func(auth model.Authorization, t txn.GetAccount_VerifyCredentials) (object.Account, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized User")
		}

		// Return as a Toot
		return user.Toot(), nil
	}
}

func PatchAccount_UpdateCredentials(serverFactory *server.Factory) func(model.Authorization, txn.PatchAccount_UpdateCredentials) (object.Account, error) {

	const location = "handler.mastodon_PatchAccount_UpdateCredentials"

	return func(auth model.Authorization, t txn.PatchAccount_UpdateCredentials) (object.Account, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized User")
		}

		// Update the User's information
		user.DisplayName = t.DisplayName
		user.Note = t.Note
		// user.ImageURL = t.ImageURL // TODO: Medium: This is not available because of Emissary's attachments
		// user.Header = t.Header // TODO: Low: This is not available because Emissary doesn't use banner images (yet)
		user.IsPublic = t.Discoverable

		if err := userService.Save(&user, "Updated via Mastodon API"); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Error saving user")
		}

		// Return updated JSON
		return user.Toot(), nil
	}
}

func GetAccount(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount) (object.Account, error) {

	const location = "handler.mastodon_GetAccount"

	return func(auth model.Authorization, t txn.GetAccount) (object.Account, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByProfileURL(t.ID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized User")
		}

		// Return updated JSON
		return user.Toot(), nil
	}
}

func GetAccount_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Statuses) ([]object.Status, error) {

	const location = "handler.mastodon_GetAccount_Statuses"

	return func(auth model.Authorization, t txn.GetAccount_Statuses) ([]object.Status, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the requested User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByProfileURL(t.ID, &user); err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized User")
		}

		// Query all posts by this user
		streamService := factory.Stream()
		streams, err := streamService.QueryByUser(user.UserID, queryExpression(t), option.MaxRows(t.Limit))

		if err != nil {
			return nil, derp.Wrap(err, location, "Error querying streams")
		}

		// TODO: HIGH: Work out how to set response headers here for additional pagination

		// Return posts as toot.Status(es)
		return sliceOfToots[model.Stream, object.Status](streams), nil
	}
}

func GetAccount_Followers(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Followers) ([]object.Account, error) {

	return func(auth model.Authorization, t txn.GetAccount_Followers) ([]object.Account, error) {

		// Emissary does not (currently?) publish followers
		return []object.Account{}, nil
	}
}

func GetAccount_Following(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Following) ([]object.Account, error) {

	return func(auth model.Authorization, t txn.GetAccount_Following) ([]object.Account, error) {

		// Emissary does not (currently?) publish following data
		return []object.Account{}, nil
	}
}

func GetAccount_FeaturedTags(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_FeaturedTags) ([]object.Tag, error) {

	return func(auth model.Authorization, t txn.GetAccount_FeaturedTags) ([]object.Tag, error) {

		// Emissary does not (currently?) publish featured tags
		return []object.Tag{}, nil
	}
}

func PostAccount_Follow(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Follow) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Follow"

	return func(auth model.Authorization, t txn.PostAccount_Follow) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Create a new "Following" record
		followingService := factory.Following()
		following := model.NewFollowing()
		following.UserID = auth.UserID
		following.URL = t.ID

		// Save the record and begin following the remote user.
		if err := followingService.Save(&following, "Created via Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error saving following")
		}

		// Return the "Following" record as a Toot
		return following.Toot(), nil
	}
}

func PostAccount_Unfollow(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unfollow) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unfollow"

	return func(auth model.Authorization, t txn.PostAccount_Unfollow) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the "Following" record
		followingService := factory.Following()
		following := model.NewFollowing()

		if err := followingService.LoadByURL(auth.UserID, t.ID, &following); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error loading following")
		}

		// Delete the "Following" record
		if err := followingService.Delete(&following, "Deleted by Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error deleting following")
		}

		return following.Toot(), nil
	}
}

func PostAccount_Block(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Block) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Block"

	return func(auth model.Authorization, t txn.PostAccount_Block) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Create a new Block record
		blockService := factory.Block()
		block := model.NewBlock()
		block.UserID = auth.UserID
		block.Type = model.BlockTypeActor
		block.Trigger = t.ID
		block.IsActive = true

		if err := blockService.Save(&block, "Created via Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error saving block")
		}

		// Return the Block record as a Toot
		return block.Toot(), nil
	}
}

func PostAccount_Unblock(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unblock) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unblock"

	return func(auth model.Authorization, t txn.PostAccount_Unblock) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Locate the block record
		blockService := factory.Block()
		block := model.NewBlock()

		if err := blockService.LoadByTrigger(auth.UserID, model.BlockTypeActor, t.ID, &block); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error loading block")
		}

		// Delete the block record
		if err := blockService.Delete(&block, "Deleted by Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error deleting block")
		}

		// Return success
		return block.Toot(), nil
	}
}

func PostAccount_Mute(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Mute) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Mute"

	return func(auth model.Authorization, t txn.PostAccount_Mute) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Create a new Block record
		blockService := factory.Block()
		block := model.NewBlock()
		block.UserID = auth.UserID
		block.Type = model.BlockTypeActor
		block.Trigger = t.ID
		block.IsActive = true

		if err := blockService.Save(&block, "Created via Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error saving block")
		}

		// Return the Block record as a Toot
		return block.Toot(), nil
	}
}

func PostAccount_Unmute(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unmute) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unmute"

	return func(auth model.Authorization, t txn.PostAccount_Unmute) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Locate the block record
		blockService := factory.Block()
		block := model.NewBlock()

		if err := blockService.LoadByTrigger(auth.UserID, model.BlockTypeActor, t.ID, &block); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error loading block")
		}

		// Delete the block record
		if err := blockService.Delete(&block, "Deleted by Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Error deleting block")
		}

		// Return success
		return block.Toot(), nil
	}
}

func PostAccount_Pin(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Pin) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Pin"

	return func(auth model.Authorization, t txn.PostAccount_Pin) (object.Relationship, error) {
		return object.Relationship{}, derp.NewBadRequestError(location, "Not implemented")
	}
}

func PostAccount_Unpin(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unpin) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unpin"

	return func(auth model.Authorization, t txn.PostAccount_Unpin) (object.Relationship, error) {
		return object.Relationship{}, derp.NewBadRequestError(location, "Not implemented")
	}
}

func PostAccount_Note(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Note) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Note"

	return func(auth model.Authorization, t txn.PostAccount_Note) (object.Relationship, error) {
		return object.Relationship{}, derp.NewBadRequestError(location, "Not implemented")
	}
}

func GetAccount_Relationships(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Relationships) ([]object.Relationship, error) {

	const location = "handler.mastodon_GetAccount_Relationships"

	return func(auth model.Authorization, t txn.GetAccount_Relationships) ([]object.Relationship, error) {
		return nil, derp.NewBadRequestError(location, "Not implemented")
	}
}

func GetAccount_FamiliarFollowers(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_FamiliarFollowers) (object.FamiliarFollowers, error) {

	const location = "handler.mastodon_GetAccount_FamiliarFollowers"

	return func(auth model.Authorization, t txn.GetAccount_FamiliarFollowers) (object.FamiliarFollowers, error) {
		return nil, derp.NewBadRequestError(location, "Not implemented")
	}
}

// https://docs.joinmastodon.org/methods/accounts/#search
func GetAccount_Search(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Search) ([]object.Account, error) {

	const location = "handler.mastodon_GetAccount_Search"

	return func(auth model.Authorization, t txn.GetAccount_Search) ([]object.Account, error) {
		return nil, derp.NewBadRequestError(location, "Not implemented")
	}
}

// https://docs.joinmastodon.org/methods/accounts/#lookup
func GetAccount_Lookup(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Lookup) (object.Account, error) {

	const location = "handler.mastodon_GetAccount_Lookup"

	return func(auth model.Authorization, t txn.GetAccount_Lookup) (object.Account, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the Account as an ActivityStream
		activityStreamsService := factory.ActivityStreams()
		document, err := activityStreamsService.Load(t.Acct)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Error loading document")
		}

		// Map the ActivityStream to a Mastodon Account
		// TODO: LOW: This should probably be moved somewhere else
		result := object.Account{
			ID:          document.ID(),
			Acct:        t.Acct,
			DisplayName: document.Name(),
			URL:         document.URL(),
		}

		// Success.
		return result, derp.NewBadRequestError(location, "Not implemented")
	}
}
