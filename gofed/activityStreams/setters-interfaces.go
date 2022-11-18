package activityStreams

import "github.com/go-fed/activity/streams/vocab"

type HasSetActor interface {
	// SetActivityStreamsActor sets the "altitude" property.
	SetActivityStreamsActor(i vocab.ActivityStreamsActorProperty)
}

type HasSetAltitude interface {
	// SetActivityStreamsAltitude sets the "altitude" property.
	SetActivityStreamsAltitude(i vocab.ActivityStreamsAltitudeProperty)
}

type HasSetAttachment interface {
	// SetActivityStreamsAttachment sets the "attachment" property.
	SetActivityStreamsAttachment(i vocab.ActivityStreamsAttachmentProperty)
}

type HasSetAttributedTo interface {
	// SetActivityStreamsAttributedTo sets the "attributedTo" property.
	SetActivityStreamsAttributedTo(i vocab.ActivityStreamsAttributedToProperty)
}

type HasSetAudience interface {
	// SetActivityStreamsAudience sets the "audience" property.
	SetActivityStreamsAudience(i vocab.ActivityStreamsAudienceProperty)
}

type HasSetBcc interface {
	// SetActivityStreamsBcc sets the "bcc" property.
	SetActivityStreamsBcc(i vocab.ActivityStreamsBccProperty)
}

type HasSetBto interface {
	// SetActivityStreamsBto sets the "bto" property.
	SetActivityStreamsBto(i vocab.ActivityStreamsBtoProperty)
}

type HasSetCc interface {
	// SetActivityStreamsCc sets the "cc" property.
	SetActivityStreamsCc(i vocab.ActivityStreamsCcProperty)
}

type HasSetContent interface {
	// SetActivityStreamsContent sets the "content" property.
	SetActivityStreamsContent(i vocab.ActivityStreamsContentProperty)
}

type HasSetContext interface {
	// SetActivityStreamsContext sets the "context" property.
	SetActivityStreamsContext(i vocab.ActivityStreamsContextProperty)
}

type HasSetDescribes interface {
	// SetActivityStreamsDescribes sets the "describes" property.
	SetActivityStreamsDescribes(i vocab.ActivityStreamsDescribesProperty)
}

type HasSetDuration interface {
	// SetActivityStreamsDuration sets the "duration" property.
	SetActivityStreamsDuration(i vocab.ActivityStreamsDurationProperty)
}

type HasSetEndTime interface {
	// SetActivityStreamsEndTime sets the "endTime" property.
	SetActivityStreamsEndTime(i vocab.ActivityStreamsEndTimeProperty)
}

type HasSetGenerator interface {
	// SetActivityStreamsGenerator sets the "generator" property.
	SetActivityStreamsGenerator(i vocab.ActivityStreamsGeneratorProperty)
}

type HasSetIcon interface {
	// SetActivityStreamsIcon sets the "icon" property.
	SetActivityStreamsIcon(i vocab.ActivityStreamsIconProperty)
}

type HasSetTags interface {
	// SetActivityStreamsImage sets the "image" property.
	SetActivityStreamsImage(i vocab.ActivityStreamsImageProperty)
}

type HasSetInReplyTo interface {
	// SetActivityStreamsInReplyTo sets the "inReplyTo" property.
	SetActivityStreamsInReplyTo(i vocab.ActivityStreamsInReplyToProperty)
}

type HasSetLikes interface {
	// SetActivityStreamsLikes sets the "likes" property.
	SetActivityStreamsLikes(i vocab.ActivityStreamsLikesProperty)
}

type HasSetLocation interface {
	// SetActivityStreamsLocation sets the "location" property.
	SetActivityStreamsLocation(i vocab.ActivityStreamsLocationProperty)
}

type HasSetMediaType interface {
	// SetActivityStreamsMediaType sets the "mediaType" property.
	SetActivityStreamsMediaType(i vocab.ActivityStreamsMediaTypeProperty)
}

type HasSetName interface {
	// SetActivityStreamsName sets the "name" property.
	SetActivityStreamsName(i vocab.ActivityStreamsNameProperty)
}

type HasSetObject interface {
	// SetActivityStreamsObject sets the "object" property.
	SetActivityStreamsObject(i vocab.ActivityStreamsObjectProperty)
}

type HasSetPreview interface {
	// SetActivityStreamsPreview sets the "preview" property.
	SetActivityStreamsPreview(i vocab.ActivityStreamsPreviewProperty)
}

type HasSetPublished interface {
	// SetActivityStreamsPublished sets the "published" property.
	SetActivityStreamsPublished(i vocab.ActivityStreamsPublishedProperty)
}

type HasSetReplies interface {
	// SetActivityStreamsReplies sets the "replies" property.
	SetActivityStreamsReplies(i vocab.ActivityStreamsRepliesProperty)
}

type HasSetShares interface {
	// SetActivityStreamsShares sets the "shares" property.
	SetActivityStreamsShares(i vocab.ActivityStreamsSharesProperty)
}

type HasSetSource interface {
	// SetActivityStreamsSource sets the "source" property.
	SetActivityStreamsSource(i vocab.ActivityStreamsSourceProperty)
}

type HasSetStartTime interface {
	// SetActivityStreamsStartTime sets the "startTime" property.
	SetActivityStreamsStartTime(i vocab.ActivityStreamsStartTimeProperty)
}

type HasSetSummary interface {
	// SetActivityStreamsSummary sets the "summary" property.
	SetActivityStreamsSummary(i vocab.ActivityStreamsSummaryProperty)
}

type HasSetTag interface {
	// SetActivityStreamsTag sets the "tag" property.
	SetActivityStreamsTag(i vocab.ActivityStreamsTagProperty)
}

type HasSetTo interface {
	// SetActivityStreamsTo sets the "to" property.
	SetActivityStreamsTo(i vocab.ActivityStreamsToProperty)
}

type HasSetUpdated interface {
	// SetActivityStreamsUpdated sets the "updated" property.
	SetActivityStreamsUpdated(i vocab.ActivityStreamsUpdatedProperty)
}

type HasSetUrl interface {
	// SetActivityStreamsUrl sets the "url" property.
	SetActivityStreamsUrl(i vocab.ActivityStreamsUrlProperty)
}
