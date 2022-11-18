package activitystreams

import "github.com/go-fed/activity/streams/vocab"

type HasActorProperty interface {
	// GetActivityStreamsActor returns the "altitude" property if it
	// exists, and nil otherwise.
	GetActivityStreamsActor() vocab.ActivityStreamsActorProperty
}

type HasAltitudeProperty interface {
	// GetActivityStreamsAltitude returns the "altitude" property if it
	// exists, and nil otherwise.
	GetActivityStreamsAltitude() vocab.ActivityStreamsAltitudeProperty
}

type HasAttachmentProperty interface {
	// GetActivityStreamsAttachment returns the "attachment" property if it
	// exists, and nil otherwise.
	GetActivityStreamsAttachment() vocab.ActivityStreamsAttachmentProperty
}

type HasAttributedToProperty interface {
	// GetActivityStreamsAttributedTo returns the "attributedTo" property if
	// it exists, and nil otherwise.
	GetActivityStreamsAttributedTo() vocab.ActivityStreamsAttributedToProperty
}

type HasAudienceProperty interface {
	// GetActivityStreamsAudience returns the "audience" property if it
	// exists, and nil otherwise.
	GetActivityStreamsAudience() vocab.ActivityStreamsAudienceProperty
}

type HasBccProperty interface {
	// GetActivityStreamsBcc returns the "bcc" property if it exists, and nil
	// otherwise.
	GetActivityStreamsBcc() vocab.ActivityStreamsBccProperty
}

type HasBtoProperty interface {
	// GetActivityStreamsBto returns the "bto" property if it exists, and nil
	// otherwise.
	GetActivityStreamsBto() vocab.ActivityStreamsBtoProperty
}

type HasCcProperty interface {
	// GetActivityStreamsCc returns the "cc" property if it exists, and nil
	// otherwise.
	GetActivityStreamsCc() vocab.ActivityStreamsCcProperty
}

type HasContentProperty interface {
	// GetActivityStreamsContent returns the "content" property if it exists,
	// and nil otherwise.
	GetActivityStreamsContent() vocab.ActivityStreamsContentProperty
}

type HasContextProperty interface {
	// GetActivityStreamsContext returns the "context" property if it exists,
	// and nil otherwise.
	GetActivityStreamsContext() vocab.ActivityStreamsContextProperty
}

type HasDescribesProperty interface {
	// GetActivityStreamsDescribes returns the "describes" property if it
	// exists, and nil otherwise.
	GetActivityStreamsDescribes() vocab.ActivityStreamsDescribesProperty
}

type HasDurationProperty interface {
	// GetActivityStreamsDuration returns the "duration" property if it
	// exists, and nil otherwise.
	GetActivityStreamsDuration() vocab.ActivityStreamsDurationProperty
}

type HasEndTimeProperty interface {
	// GetActivityStreamsEndTime returns the "endTime" property if it exists,
	// and nil otherwise.
	GetActivityStreamsEndTime() vocab.ActivityStreamsEndTimeProperty
}

type HasGeneratorProperty interface {
	// GetActivityStreamsGenerator returns the "generator" property if it
	// exists, and nil otherwise.
	GetActivityStreamsGenerator() vocab.ActivityStreamsGeneratorProperty
}

type HasIconProperty interface {
	// GetActivityStreamsIcon returns the "icon" property if it exists, and
	// nil otherwise.
	GetActivityStreamsIcon() vocab.ActivityStreamsIconProperty
}

type HasImageProperty interface {
	// GetActivityStreamsImage returns the "image" property if it exists, and
	// nil otherwise.
	GetActivityStreamsImage() vocab.ActivityStreamsImageProperty
}

type HasInReplyToProperty interface {
	// GetActivityStreamsInReplyTo returns the "inReplyTo" property if it
	// exists, and nil otherwise.
	GetActivityStreamsInReplyTo() vocab.ActivityStreamsInReplyToProperty
}

type HasLikesProperty interface {
	// GetActivityStreamsLikes returns the "likes" property if it exists, and
	// nil otherwise.
	GetActivityStreamsLikes() vocab.ActivityStreamsLikesProperty
}

type HasLocationProperty interface {
	// GetActivityStreamsLocation returns the "location" property if it
	// exists, and nil otherwise.
	GetActivityStreamsLocation() vocab.ActivityStreamsLocationProperty
}

type HasMediaTypeProperty interface {
	// GetActivityStreamsMediaType returns the "mediaType" property if it
	// exists, and nil otherwise.
	GetActivityStreamsMediaType() vocab.ActivityStreamsMediaTypeProperty
}

type HasNameProperty interface {
	// GetActivityStreamsName returns the "name" property if it exists, and
	// nil otherwise.
	GetActivityStreamsName() vocab.ActivityStreamsNameProperty
}

type HasObjectProperty interface {
	// GetActivityStreamsObject returns the "object" property if it exists,
	// and nil otherwise.
	GetActivityStreamsObject() vocab.ActivityStreamsObjectProperty
}

type HasPreviewProperty interface {
	// GetActivityStreamsPreview returns the "preview" property if it exists,
	// and nil otherwise.
	GetActivityStreamsPreview() vocab.ActivityStreamsPreviewProperty
}

type HasPublishedProperty interface {
	// GetActivityStreamsPublished returns the "published" property if it
	// exists, and nil otherwise.
	GetActivityStreamsPublished() vocab.ActivityStreamsPublishedProperty
}

type HasRepliesProperty interface {
	// GetActivityStreamsReplies returns the "replies" property if it exists,
	// and nil otherwise.
	GetActivityStreamsReplies() vocab.ActivityStreamsRepliesProperty
}

type HasSharesProperty interface {
	// GetActivityStreamsShares returns the "shares" property if it exists,
	// and nil otherwise.
	GetActivityStreamsShares() vocab.ActivityStreamsSharesProperty
}

type HasSourceProperty interface {
	// GetActivityStreamsSource returns the "source" property if it exists,
	// and nil otherwise.
	GetActivityStreamsSource() vocab.ActivityStreamsSourceProperty
}

type HasStartTimeProperty interface {
	// GetActivityStreamsStartTime returns the "startTime" property if it
	// exists, and nil otherwise.
	GetActivityStreamsStartTime() vocab.ActivityStreamsStartTimeProperty
}

type HasSummaryProperty interface {
	// GetActivityStreamsSummary returns the "summary" property if it exists,
	// and nil otherwise.
	GetActivityStreamsSummary() vocab.ActivityStreamsSummaryProperty
}

type HasTagProperty interface {
	// GetActivityStreamsTag returns the "tag" property if it exists, and nil
	// otherwise.
	GetActivityStreamsTag() vocab.ActivityStreamsTagProperty
}

type HasToProperty interface {
	// GetActivityStreamsTo returns the "to" property if it exists, and nil
	// otherwise.
	GetActivityStreamsTo() vocab.ActivityStreamsToProperty
}

type HasUpdatedProperty interface {
	// GetActivityStreamsUpdated returns the "updated" property if it exists,
	// and nil otherwise.
	GetActivityStreamsUpdated() vocab.ActivityStreamsUpdatedProperty
}

type HasUrlProperty interface {
	// GetActivityStreamsUrl returns the "url" property if it exists, and nil
	// otherwise.
	GetActivityStreamsUrl() vocab.ActivityStreamsUrlProperty
}
