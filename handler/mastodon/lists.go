package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// https://docs.joinmastodon.org/methods/lists/

// https://docs.joinmastodon.org/methods/lists/#get
func GetLists(serverFactory *server.Factory) func(model.Authorization, txn.GetLists) ([]object.List, error) {

	const location = "handler.mastodon.GetLists"

	return func(auth model.Authorization, t txn.GetLists) ([]object.List, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Get all folders
		folderService := factory.Folder()
		folders, err := folderService.QueryByUserID(auth.UserID)

		if err != nil {
			return nil, derp.Wrap(err, location, "Error querying database")
		}

		return sliceOfToots[model.Folder, object.List](folders), nil
	}
}

// https://docs.joinmastodon.org/methods/lists/#get-one
func GetList(serverFactory *server.Factory) func(model.Authorization, txn.GetList) (object.List, error) {

	const location = "handler.mastodon.GetList"

	return func(auth model.Authorization, t txn.GetList) (object.List, error) {

		// Collect Arguments
		folderID, err := primitive.ObjectIDFromHex(t.ID)

		if err != nil {
			return object.List{}, derp.Wrap(err, location, "Invalid Folder ID", t.ID)
		}

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.List{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Get the Folder from the Database
		folderService := factory.Folder()
		folder := model.NewFolder()

		if err := folderService.LoadByID(auth.UserID, folderID, &folder); err != nil {
			return object.List{}, derp.Wrap(err, location, "Error loading folder")
		}

		// Reply with a Toot!
		return folder.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/lists/#create
func PostList(serverFactory *server.Factory) func(model.Authorization, txn.PostList) (object.List, error) {

	const location = "handler.mastodon.PostList"

	return func(auth model.Authorization, t txn.PostList) (object.List, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.List{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Create a New Folder
		folder := model.NewFolder()
		folder.UserID = auth.UserID
		folder.Label = t.Title

		// Save it to the database
		folderService := factory.Folder()
		if err := folderService.Save(&folder, "Created via Mastodon API"); err != nil {
			return object.List{}, derp.Wrap(err, location, "Error saving folder")
		}

		// Return a Toot!
		return folder.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/lists/#update
func PutList(serverFactory *server.Factory) func(model.Authorization, txn.PutList) (object.List, error) {

	const location = "handler.mastodon.PutList"

	return func(auth model.Authorization, t txn.PutList) (object.List, error) {

		// Collect Arguments
		folderID, err := primitive.ObjectIDFromHex(t.ID)

		if err != nil {
			return object.List{}, derp.Wrap(err, location, "Invalid Folder ID", t.ID)
		}

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return object.List{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Get the Folder from the Database
		folderService := factory.Folder()
		folder := model.NewFolder()

		if err := folderService.LoadByID(auth.UserID, folderID, &folder); err != nil {
			return object.List{}, derp.Wrap(err, location, "Error loading folder")
		}

		// Update Folder Data
		folder.Label = t.Title

		// Save it to the database
		if err := folderService.Save(&folder, "Created via Mastodon API"); err != nil {
			return object.List{}, derp.Wrap(err, location, "Error saving folder")
		}

		// Return a Toot!
		return folder.Toot(), nil
	}
}

func DeleteList(serverFactory *server.Factory) func(model.Authorization, txn.DeleteList) (struct{}, error) {

	const location = "handler.mastodon.DeleteList"

	return func(auth model.Authorization, t txn.DeleteList) (struct{}, error) {

		// Collect Arguments
		folderID, err := primitive.ObjectIDFromHex(t.ID)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Invalid Folder ID", t.ID)
		}

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Load the Folder from the Database
		folderService := factory.Folder()
		folder := model.NewFolder()

		if err := folderService.LoadByID(auth.UserID, folderID, &folder); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error loading folder")
		}

		// Delete the Folder
		if err := folderService.Delete(&folder, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error deleting folder")
		}

		// Return a successful response
		return struct{}{}, nil
	}
}

// https://docs.joinmastodon.org/methods/lists/#accounts
func GetList_Accounts(serverFactory *server.Factory) func(model.Authorization, txn.GetList_Accounts) ([]object.Account, error) {

	const location = "handler.mastodon.GetList_Accounts"

	return func(auth model.Authorization, t txn.GetList_Accounts) ([]object.Account, error) {

		// Collect Arguments
		folderID, err := primitive.ObjectIDFromHex(t.ID)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid Folder ID", t.ID)
		}

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Query all Following records in this Folder
		followingService := factory.Following()
		criteria := queryExpression(t)
		followingSummaries, err := followingService.QueryByFolderAndExp(auth.UserID, folderID, criteria)

		// Convert the results to a slice of objects
		result := slice.Map(followingSummaries, func(following model.FollowingSummary) object.Account {

			return object.Account{
				ID:     following.FollowingID.Hex(),
				URL:    following.URL,
				Avatar: following.ImageURL,
			}
		})

		// Return results to caller
		return result, nil
	}
}

func PostList_Accounts(serverFactory *server.Factory) func(model.Authorization, txn.PostList_Accounts) (struct{}, error) {

	return func(model.Authorization, txn.PostList_Accounts) (struct{}, error) {
		return struct{}{}, derp.NewBadRequestError("handler.mastodon.PostListAccounts", "Not Implemented")
	}
}
func DeleteList_Accounts(serverFactory *server.Factory) func(model.Authorization, txn.DeleteList_Accounts) (struct{}, error) {

	return func(model.Authorization, txn.DeleteList_Accounts) (struct{}, error) {
		return struct{}{}, derp.NewBadRequestError("handler.mastodon.PostListAccounts", "Not Implemented")
	}
}
