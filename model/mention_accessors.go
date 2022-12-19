package model

import "go.mongodb.org/mongo-driver/bson/primitive"

func (mention *Mention) GetObjectID(name string) primitive.ObjectID {

	switch name {
	case "mentionId":
		return mention.MentionID
	case "streamId":
		return mention.StreamID
	}
	return primitive.NilObjectID
}

func (mention *Mention) GetString(name string) string {
	switch name {
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
