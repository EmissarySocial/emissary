package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*********************************
 * Getter Methods
 *********************************/

func (mention *Mention) GetStringOK(name string) (string, bool) {
	switch name {

	case "mentionId":
		return mention.MentionID.Hex(), true

	case "streamId":
		return mention.StreamID.Hex(), true

	case "originUrl":
		return mention.OriginURL, true

	case "authorName":
		return mention.AuthorName, true

	case "authorEmail":
		return mention.AuthorEmail, true

	case "authorWebsiteUrl":
		return mention.AuthorWebsiteURL, true

	case "authorPhotoUrl":
		return mention.AuthorPhotoURL, true

	case "authorStatus":
		return mention.AuthorStatus, true

	case "entryName":
		return mention.EntryName, true

	case "entrySummary":
		return mention.EntrySummary, true

	case "entryPhotoUrl":
		return mention.EntryPhotoURL, true
	}
	return "", false
}

/*********************************
 * Setter Methods
 *********************************/

func (mention *Mention) SetStringOK(name string, value string) bool {
	switch name {

	case "mentionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			mention.MentionID = objectID
			return true
		}

	case "streamId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			mention.StreamID = objectID
			return true
		}

	case "originUrl":
		mention.OriginURL = value
		return true

	case "authorName":
		mention.AuthorName = value
		return true

	case "authorEmail":
		mention.AuthorEmail = value
		return true

	case "authorWebsiteUrl":
		mention.AuthorWebsiteURL = value
		return true

	case "authorPhotoUrl":
		mention.AuthorPhotoURL = value
		return true

	case "authorStatus":
		mention.AuthorStatus = value
		return true

	case "entryName":
		mention.EntryName = value
		return true

	case "entrySummary":
		mention.EntrySummary = value
		return true

	case "entryPhotoUrl":
		mention.EntryPhotoURL = value
		return true

	}

	return false
}
