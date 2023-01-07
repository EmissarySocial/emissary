package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*********************************
 * Getter Methods
 *********************************/

func (mention *Mention) GetString(name string) string {
	switch name {

	case "mentionId":
		return mention.MentionID.Hex()
	case "streamId":
		return mention.StreamID.Hex()
	case "originUrl":
		return mention.OriginURL
	case "authorName":
		return mention.AuthorName
	case "authorEmail":
		return mention.AuthorEmail
	case "authorWebsiteUrl":
		return mention.AuthorWebsiteURL
	case "authorPhotoUrl":
		return mention.AuthorPhotoURL
	case "authorStatus":
		return mention.AuthorStatus
	case "entryName":
		return mention.EntryName
	case "entrySummary":
		return mention.EntrySummary
	case "entryPhotoUrl":
		return mention.EntryPhotoURL
	}
	return ""
}

/*********************************
 * Setter Methods
 *********************************/

func (mention *Mention) SetString(name string, value string) bool {
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
