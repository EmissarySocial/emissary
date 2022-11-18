package activityStreams

import "github.com/go-fed/activity/streams/vocab"

func GetActor(item vocab.Type) vocab.ActivityStreamsActorProperty {
	if i, ok := item.(HasActorProperty); ok {
		return i.GetActivityStreamsActor()
	}
	return nil
}

func GetAltitude(item vocab.Type) vocab.ActivityStreamsAltitudeProperty {
	if i, ok := item.(HasAltitudeProperty); ok {
		return i.GetActivityStreamsAltitude()
	}
	return nil
}

func GetAttachment(item vocab.Type) vocab.ActivityStreamsAttachmentProperty {
	if i, ok := item.(HasAttachmentProperty); ok {
		return i.GetActivityStreamsAttachment()
	}
	return nil
}

func GetAttributedTo(item vocab.Type) vocab.ActivityStreamsAttributedToProperty {
	if i, ok := item.(HasAttributedToProperty); ok {
		return i.GetActivityStreamsAttributedTo()
	}
	return nil
}

func GetAudience(item vocab.Type) vocab.ActivityStreamsAudienceProperty {
	if i, ok := item.(HasAudienceProperty); ok {
		return i.GetActivityStreamsAudience()
	}
	return nil
}

func GetBcc(item vocab.Type) vocab.ActivityStreamsBccProperty {
	if i, ok := item.(HasBccProperty); ok {
		return i.GetActivityStreamsBcc()
	}
	return nil
}

func GetBto(item vocab.Type) vocab.ActivityStreamsBtoProperty {
	if i, ok := item.(HasBtoProperty); ok {
		return i.GetActivityStreamsBto()
	}
	return nil
}

func GetCc(item vocab.Type) vocab.ActivityStreamsCcProperty {
	if i, ok := item.(HasCcProperty); ok {
		return i.GetActivityStreamsCc()
	}
	return nil
}

func GetContent(item vocab.Type) vocab.ActivityStreamsContentProperty {
	if i, ok := item.(HasContentProperty); ok {
		return i.GetActivityStreamsContent()
	}
	return nil
}

func GetContext(item vocab.Type) vocab.ActivityStreamsContextProperty {
	if i, ok := item.(HasContextProperty); ok {
		return i.GetActivityStreamsContext()
	}
	return nil
}

func GetDescribes(item vocab.Type) vocab.ActivityStreamsDescribesProperty {
	if i, ok := item.(HasDescribesProperty); ok {
		return i.GetActivityStreamsDescribes()
	}
	return nil
}

func GetDuration(item vocab.Type) vocab.ActivityStreamsDurationProperty {
	if i, ok := item.(HasDurationProperty); ok {
		return i.GetActivityStreamsDuration()
	}
	return nil
}

func GetEndTime(item vocab.Type) vocab.ActivityStreamsEndTimeProperty {
	if i, ok := item.(HasEndTimeProperty); ok {
		return i.GetActivityStreamsEndTime()
	}
	return nil
}

func GetGenerator(item vocab.Type) vocab.ActivityStreamsGeneratorProperty {
	if i, ok := item.(HasGeneratorProperty); ok {
		return i.GetActivityStreamsGenerator()
	}
	return nil
}

func GetIcon(item vocab.Type) vocab.ActivityStreamsIconProperty {
	if i, ok := item.(HasIconProperty); ok {
		return i.GetActivityStreamsIcon()
	}
	return nil
}

func GetImage(item vocab.Type) vocab.ActivityStreamsImageProperty {
	if i, ok := item.(HasImageProperty); ok {
		return i.GetActivityStreamsImage()
	}
	return nil
}

func GetInReplyTo(item vocab.Type) vocab.ActivityStreamsInReplyToProperty {
	if i, ok := item.(HasInReplyToProperty); ok {
		return i.GetActivityStreamsInReplyTo()
	}
	return nil
}

func GetLikes(item vocab.Type) vocab.ActivityStreamsLikesProperty {
	if i, ok := item.(HasLikesProperty); ok {
		return i.GetActivityStreamsLikes()
	}
	return nil
}

func GetLocation(item vocab.Type) vocab.ActivityStreamsLocationProperty {
	if i, ok := item.(HasLocationProperty); ok {
		return i.GetActivityStreamsLocation()
	}
	return nil
}

func GetMediaType(item vocab.Type) vocab.ActivityStreamsMediaTypeProperty {
	if i, ok := item.(HasMediaTypeProperty); ok {
		return i.GetActivityStreamsMediaType()
	}
	return nil
}

func GetObject(item vocab.Type) vocab.ActivityStreamsObjectProperty {
	if i, ok := item.(HasObjectProperty); ok {
		return i.GetActivityStreamsObject()
	}
	return nil
}

func GetPreview(item vocab.Type) vocab.ActivityStreamsPreviewProperty {
	if i, ok := item.(HasPreviewProperty); ok {
		return i.GetActivityStreamsPreview()
	}
	return nil
}

func GetPublished(item vocab.Type) vocab.ActivityStreamsPublishedProperty {
	if i, ok := item.(HasPublishedProperty); ok {
		return i.GetActivityStreamsPublished()
	}
	return nil
}

func GetReplies(item vocab.Type) vocab.ActivityStreamsRepliesProperty {
	if i, ok := item.(HasRepliesProperty); ok {
		return i.GetActivityStreamsReplies()
	}
	return nil
}

func GetShares(item vocab.Type) vocab.ActivityStreamsSharesProperty {
	if i, ok := item.(HasSharesProperty); ok {
		return i.GetActivityStreamsShares()
	}
	return nil
}

func GetSource(item vocab.Type) vocab.ActivityStreamsSourceProperty {
	if i, ok := item.(HasSourceProperty); ok {
		return i.GetActivityStreamsSource()
	}
	return nil
}

func GetStartTime(item vocab.Type) vocab.ActivityStreamsStartTimeProperty {
	if i, ok := item.(HasStartTimeProperty); ok {
		return i.GetActivityStreamsStartTime()
	}
	return nil
}

func GetSummary(item vocab.Type) vocab.ActivityStreamsSummaryProperty {
	if i, ok := item.(HasSummaryProperty); ok {
		return i.GetActivityStreamsSummary()
	}
	return nil
}

func GetTag(item vocab.Type) vocab.ActivityStreamsTagProperty {
	if i, ok := item.(HasTagProperty); ok {
		return i.GetActivityStreamsTag()
	}
	return nil
}

func GetTo(item vocab.Type) vocab.ActivityStreamsToProperty {
	if i, ok := item.(HasToProperty); ok {
		return i.GetActivityStreamsTo()
	}
	return nil
}

func GetUpdated(item vocab.Type) vocab.ActivityStreamsUpdatedProperty {
	if i, ok := item.(HasUpdatedProperty); ok {
		return i.GetActivityStreamsUpdated()
	}
	return nil
}

func GetUrl(item vocab.Type) vocab.ActivityStreamsUrlProperty {
	if i, ok := item.(HasUrlProperty); ok {
		return i.GetActivityStreamsUrl()
	}
	return nil
}
