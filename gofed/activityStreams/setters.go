package activityStreams

import "github.com/go-fed/activity/streams/vocab"

func SetActor(item vocab.Type, value vocab.ActivityStreamsActorProperty) {
	if i, ok := item.(HasSetActor); ok {
		i.SetActivityStreamsActor(value)
	}
}

func SetAltitude(item vocab.Type, value vocab.ActivityStreamsAltitudeProperty) {
	if i, ok := item.(HasSetAltitude); ok {
		i.SetActivityStreamsAltitude(value)
	}
}

func SetAttachment(item vocab.Type, value vocab.ActivityStreamsAttachmentProperty) {
	if i, ok := item.(HasSetAttachment); ok {
		i.SetActivityStreamsAttachment(value)
	}
}

func SetAtributedTo(item vocab.Type, value vocab.ActivityStreamsAttributedToProperty) {
	if i, ok := item.(HasSetAttributedTo); ok {
		i.SetActivityStreamsAttributedTo(value)
	}
}

func SetAudience(item vocab.Type, value vocab.ActivityStreamsAudienceProperty) {
	if i, ok := item.(HasSetAudience); ok {
		i.SetActivityStreamsAudience(value)
	}
}

func SetBcc(item vocab.Type, value vocab.ActivityStreamsBccProperty) {
	if i, ok := item.(HasSetBcc); ok {
		i.SetActivityStreamsBcc(value)
	}
}

func SetBto(item vocab.Type, value vocab.ActivityStreamsBtoProperty) {
	if i, ok := item.(HasSetBto); ok {
		i.SetActivityStreamsBto(value)
	}
}

func SetCc(item vocab.Type, value vocab.ActivityStreamsCcProperty) {
	if i, ok := item.(HasSetCc); ok {
		i.SetActivityStreamsCc(value)
	}
}

func SetContent(item vocab.Type, value vocab.ActivityStreamsContentProperty) {
	if i, ok := item.(HasSetContent); ok {
		i.SetActivityStreamsContent(value)
	}
}

func SetContext(item vocab.Type, value vocab.ActivityStreamsContextProperty) {
	if i, ok := item.(HasSetContext); ok {
		i.SetActivityStreamsContext(value)
	}
}

func SetDescribes(item vocab.Type, value vocab.ActivityStreamsDescribesProperty) {
	if i, ok := item.(HasSetDescribes); ok {
		i.SetActivityStreamsDescribes(value)
	}
}

func SetDuration(item vocab.Type, value vocab.ActivityStreamsDurationProperty) {
	if i, ok := item.(HasSetDuration); ok {
		i.SetActivityStreamsDuration(value)
	}
}

func SetEndTime(item vocab.Type, value vocab.ActivityStreamsEndTimeProperty) {
	if i, ok := item.(HasSetEndTime); ok {
		i.SetActivityStreamsEndTime(value)
	}
}

func SetGenerator(item vocab.Type, value vocab.ActivityStreamsGeneratorProperty) {
	if i, ok := item.(HasSetGenerator); ok {
		i.SetActivityStreamsGenerator(value)
	}
}

func SetIcon(item vocab.Type, value vocab.ActivityStreamsIconProperty) {
	if i, ok := item.(HasSetIcon); ok {
		i.SetActivityStreamsIcon(value)
	}
}

func SetImage(item vocab.Type, value vocab.ActivityStreamsImageProperty) {
	if i, ok := item.(HasSetTags); ok {
		i.SetActivityStreamsImage(value)
	}
}

func SetInReplyTo(item vocab.Type, value vocab.ActivityStreamsInReplyToProperty) {
	if i, ok := item.(HasSetInReplyTo); ok {
		i.SetActivityStreamsInReplyTo(value)
	}
}

func SetLikes(item vocab.Type, value vocab.ActivityStreamsLikesProperty) {
	if i, ok := item.(HasSetLikes); ok {
		i.SetActivityStreamsLikes(value)
	}
}

func SetLocation(item vocab.Type, value vocab.ActivityStreamsLocationProperty) {
	if i, ok := item.(HasSetLocation); ok {
		i.SetActivityStreamsLocation(value)
	}
}

func SetMediaType(item vocab.Type, value vocab.ActivityStreamsMediaTypeProperty) {
	if i, ok := item.(HasSetMediaType); ok {
		i.SetActivityStreamsMediaType(value)
	}
}

func SetName(item vocab.Type, value vocab.ActivityStreamsNameProperty) {
	if i, ok := item.(HasSetName); ok {
		i.SetActivityStreamsName(value)
	}
}

func SetObject(item vocab.Type, value vocab.ActivityStreamsObjectProperty) {
	if i, ok := item.(HasSetObject); ok {
		i.SetActivityStreamsObject(value)
	}
}

func SetPreview(item vocab.Type, value vocab.ActivityStreamsPreviewProperty) {
	if i, ok := item.(HasSetPreview); ok {
		i.SetActivityStreamsPreview(value)
	}
}

func SetPublished(item vocab.Type, value vocab.ActivityStreamsPublishedProperty) {
	if i, ok := item.(HasSetPublished); ok {
		i.SetActivityStreamsPublished(value)
	}
}

func SetReplies(item vocab.Type, value vocab.ActivityStreamsRepliesProperty) {
	if i, ok := item.(HasSetReplies); ok {
		i.SetActivityStreamsReplies(value)
	}
}

func SetShares(item vocab.Type, value vocab.ActivityStreamsSharesProperty) {
	if i, ok := item.(HasSetShares); ok {
		i.SetActivityStreamsShares(value)
	}
}

func SetSource(item vocab.Type, value vocab.ActivityStreamsSourceProperty) {
	if i, ok := item.(HasSetSource); ok {
		i.SetActivityStreamsSource(value)
	}
}

func SetStartTime(item vocab.Type, value vocab.ActivityStreamsStartTimeProperty) {
	if i, ok := item.(HasSetStartTime); ok {
		i.SetActivityStreamsStartTime(value)
	}
}

func SetSummary(item vocab.Type, value vocab.ActivityStreamsSummaryProperty) {
	if i, ok := item.(HasSetSummary); ok {
		i.SetActivityStreamsSummary(value)
	}
}

func SetTag(item vocab.Type, value vocab.ActivityStreamsTagProperty) {
	if i, ok := item.(HasSetTag); ok {
		i.SetActivityStreamsTag(value)
	}
}

func SetTo(item vocab.Type, value vocab.ActivityStreamsToProperty) {
	if i, ok := item.(HasSetTo); ok {
		i.SetActivityStreamsTo(value)
	}
}

func SetUpdated(item vocab.Type, value vocab.ActivityStreamsUpdatedProperty) {
	if i, ok := item.(HasSetUpdated); ok {
		i.SetActivityStreamsUpdated(value)
	}
}

func SetUrl(item vocab.Type, value vocab.ActivityStreamsUrlProperty) {
	if i, ok := item.(HasSetUrl); ok {
		i.SetActivityStreamsUrl(value)
	}
}
