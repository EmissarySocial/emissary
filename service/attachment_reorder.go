package service

import (
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Reorder gives the Attachments named in `order`, in that order, the ranks they already hold
// between them, and leaves every other Attachment in the category where it is. An ID that is not
// in the category is skipped, and ranks that are tied or unset are renumbered in their current
// order before any are traded.
func (service *Attachment) Reorder(session data.Session, objectType string, objectID primitive.ObjectID, category string, order []primitive.ObjectID, note string) error {

	const location = "service.Attachment.Reorder"

	// Load every Attachment in the category before saving any, because the trade reads all of their ranks
	attachments, err := service.rankedByCategory(session, objectType, objectID, category)

	if err != nil {
		return derp.Wrap(err, location, "Loading attachments", objectType, objectID, category)
	}

	// Save each Attachment whose rank changed
	ranks := reorderRanks(attachments, order)

	for index := range attachments {

		attachment := &attachments[index]
		newRank := ranks[attachment.AttachmentID]

		if newRank == attachment.Rank {
			continue
		}

		attachment.Rank = newRank

		if err := service.Save(session, attachment, note); err != nil {
			return derp.Wrap(err, location, "Saving attachment", attachment.AttachmentID)
		}
	}

	// Everything in its right place
	return nil
}

// rankedByCategory returns every Attachment on the object in the category, in rank order.
// An empty category means every category.
func (service *Attachment) rankedByCategory(session data.Session, objectType string, objectID primitive.ObjectID, category string) ([]model.Attachment, error) {

	attachments, err := service.Range(session, categoryCriteria(objectType, objectID, category), option.SortAsc("rank"))

	if err != nil {
		return nil, err
	}

	return sortByRank(slices.Collect(attachments)), nil
}

// reorderRanks returns the rank each Attachment should hold so that the ones named in `order`
// take, in that order, the ranks they already hold between them. Attachments not named keep
// their ranks, and an ID that names no Attachment is skipped.
func reorderRanks(attachments []model.Attachment, order []primitive.ObjectID) map[primitive.ObjectID]int {

	ranks := currentRanks(attachments)

	// Collect the named Attachments that exist, once each, in the order named
	named := make([]primitive.ObjectID, 0, len(order))
	seen := make(map[primitive.ObjectID]bool, len(order))

	for _, attachmentID := range order {

		if _, exists := ranks[attachmentID]; !exists {
			continue
		}

		if seen[attachmentID] {
			continue
		}

		seen[attachmentID] = true
		named = append(named, attachmentID)
	}

	// Hand out the ranks they hold between them, lowest first, in the order named
	held := make([]int, 0, len(named))

	for _, attachmentID := range named {
		held = append(held, ranks[attachmentID])
	}

	slices.Sort(held)

	for index, attachmentID := range named {
		ranks[attachmentID] = held[index]
	}

	return ranks
}

// currentRanks returns each Attachment's rank, renumbered 1 to n in the order given when any
// rank is unset (below 1) or shared, because a rank like that cannot be traded.
func currentRanks(attachments []model.Attachment) map[primitive.ObjectID]int {

	ranks := make(map[primitive.ObjectID]int, len(attachments))

	if ranksAreDistinct(attachments) {
		for _, attachment := range attachments {
			ranks[attachment.AttachmentID] = attachment.Rank
		}
		return ranks
	}

	// Every file stored before ranks were assigned holds 0, so this is the common case once
	for index, attachment := range attachments {
		ranks[attachment.AttachmentID] = index + 1
	}

	return ranks
}

// ranksAreDistinct returns TRUE when every Attachment holds a rank of 1 or more that no other holds
func ranksAreDistinct(attachments []model.Attachment) bool {

	seen := make(map[int]bool, len(attachments))

	for _, attachment := range attachments {

		if attachment.Rank < 1 {
			return false
		}

		if seen[attachment.Rank] {
			return false
		}

		seen[attachment.Rank] = true
	}

	return true
}
