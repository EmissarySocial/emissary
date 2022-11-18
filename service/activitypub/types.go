package activitypub

/*********************************
 * Activity Types
 * https://www.w3.org/TR/activitystreams-vocabulary/#object-types
 *********************************/

// ActivityTypeAccept description is TBD
const ActivityTypeAccept = "Accept"

// ActivityTypeAdd description is TBD
const ActivityTypeAdd = "Add"

// ActivityTypeAnnounce description is TBD
const ActivityTypeAnnounce = "Announce"

// ActivityTypeArrive description is TBD
const ActivityTypeArrive = "Arrive"

// ActivityTypeBlock description is TBD
const ActivityTypeBlock = "Block"

// ActivityTypeCreate description is TBD
const ActivityTypeCreate = "Create"

// ActivityTypeDelete description is TBD
const ActivityTypeDelete = "Delete"

// ActivityTypeDislike description is TBD
const ActivityTypeDislike = "Dislike"

// ActivityTypeFlag description is TBD
const ActivityTypeFlag = "Flag"

// ActivityTypeFollow description is TBD
const ActivityTypeFollow = "Follow"

// ActivityTypeIgnore description is TBD
const ActivityTypeIgnore = "Ignore"

// ActivityTypeInvite description is TBD
const ActivityTypeInvite = "Invite"

// ActivityTypeJoin description is TBD
const ActivityTypeJoin = "Join"

// ActivityTypeLeave description is TBD
const ActivityTypeLeave = "Leave"

// ActivityTypeLike description is TBD
const ActivityTypeLike = "Like"

// ActivityTypeListen description is TBD
const ActivityTypeListen = "Listen"

// ActivityTypeMove description is TBD
const ActivityTypeMove = "Move"

// ActivityTypeOffer description is TBD
const ActivityTypeOffer = "Offer"

// ActivityTypeQuestion description is TBD
const ActivityTypeQuestion = "Question"

// ActivityTypeReject description is TBD
const ActivityTypeReject = "Reject"

// ActivityTypeRead description is TBD
const ActivityTypeRead = "Read"

// ActivityTypeRemove description is TBD
const ActivityTypeRemove = "Remove"

// ActivityTypeTentativeReject description is TBD
const ActivityTypeTentativeReject = "TentativeReject"

// ActivityTypeTentativeAccept description is TBD
const ActivityTypeTentativeAccept = "TentativeAccept"

// ActivityTypeTravel description is TBD
const ActivityTypeTravel = "Travel"

// ActivityTypeUndo description is TBD
const ActivityTypeUndo = "Undo"

// ActivityTypeUpdate description is TBD
const ActivityTypeUpdate = "Update"

// ActivityTypeView description is TBD
const ActivityTypeView = "View"

/*********************************
 * Actor Types
 * https://www.w3.org/TR/activitystreams-vocabulary/#actor-types
 *********************************/

const ActorTypeApplication = "Application"

const ActorTypeGroup = "Group"

const ActorTypeOrganization = "Organization"

const ActorTypePerson = "Person"

const ActorTypeService = "Service"

const ItemTypeInbox = "inbox"

const ItemTypeOutbox = "outbox"

const ItemTypeFollowers = "followers"

const ItemTypeFollowing = "following"

const ItemTypeLiked = "liked"

/*********************************
 * Link Types
 * TODO: Need a source for these.
 *********************************/

// LinkTypeMention is a specialized Link that represents an @mention.
const LinkTypeMention = "Mention"

/*********************************
 * Object Types
 * https://www.w3.org/TR/activitystreams-vocabulary/#object-types
 *********************************/

// ObjectTypeArticle represents any kind of multi-paragraph written work.
const ObjectTypeArticle = "Article"

// ObjectTypeAudio represents an audio document of any kind.
const ObjectTypeAudio = "Audio"

// ObjectTypeDocument represents a document of any kind.
const ObjectTypeDocument = "Document"

// ObjectTypeEvent represents any kind of event.
const ObjectTypeEvent = "Event"

// ObjectTypeImage represents an image document of any kind
const ObjectTypeImage = "Image"

// ObjectTypeMention ???
const ObjectTypeMention = "Mention"

// ObjectTypeNote represents a short written work typically less than a single paragraph in length.
const ObjectTypeNote = "Note"

// ObjectTypePage represents a Web Page.
const ObjectTypePage = "Page"

// ObjectTypePlace represents a logical or physical location. See 5.3 Representing Places for additional information.
const ObjectTypePlace = "Place"

// ObjectTypeProfile is a content object that describes another Object, typically used to describe Actor Type objects. The describes property is used to reference the object being described by the profile.
const ObjectTypeProfile = "Profile"

// ObjectTypeRelationship describes a relationship between two individuals. The subject and object properties are used to identify the connected individuals.
const ObjectTypeRelationship = "Relationship"

// ObjectTypeTombstone represents a content object that has been deleted. It can be used in Collections to signify that there used to be an object at this position, but it has been deleted.
const ObjectTypeTombstone = "Tombstone"

// ObjectTypeVideo represents a video document of any kind.
const ObjectTypeVideo = "Video"

/*********************************
 * Property Types
 * https://www.w3.org/TR/activitystreams-vocabulary/#properties
 *********************************/

// PropertySubject description is TBD
const PropertySubject = "subject"

// PropertyRelationship description is TBD
const PropertyRelationship = "relationship"

// PropertyActor description is TBD
const PropertyActor = "actor"

// PropertyAttributedTo description is TBD
const PropertyAttributedTo = "attributedTo"

// PropertyAttachment description is TBD
const PropertyAttachment = "attachment"

// PropertyAttachments description is TBD
const PropertyAttachments = "attachments"

// PropertyAuthor description is TBD
const PropertyAuthor = "author"

// PropertyBcc description is TBD
const PropertyBcc = "bcc"

// PropertyBTo description is TBD
const PropertyBTo = "bto"

// PropertyCc description is TBD
const PropertyCc = "cc"

// PropertyContext description is TBD
const PropertyContext = "@context"

// PropertyCurrent description is TBD
const PropertyCurrent = "current"

// PropertyFirst description is TBD
const PropertyFirst = "first"

// PropertyGenerator description is TBD
const PropertyGenerator = "generator"

// PropertyID description is TBD
const PropertyID = "id"

// PropertyIcon description is TBD
const PropertyIcon = "icon"

// PropertyImage description is TBD
const PropertyImage = "image"

// PropertyInReplyTo description is TBD
const PropertyInReplyTo = "inReplyTo"

// PropertyItems description is TBD
const PropertyItems = "items"

// PropertyInstrument description is TBD
const PropertyInstrument = "instrument"

// PropertyOrderedItems description is TBD
const PropertyOrderedItems = "orderedItems"

// PropertyLast description is TBD
const PropertyLast = "last"

// PropertyLocation description is TBD
const PropertyLocation = "location"

// PropertyNext description is TBD
const PropertyNext = "next"

// PropertyObject description is TBD
const PropertyObject = "object"

// PropertyOneOf description is TBD
const PropertyOneOf = "oneOf"

// PropertyAnyOf description is TBD
const PropertyAnyOf = "anyOf"

// PropertyOrigin description is TBD
const PropertyOrigin = "origin"

// PropertyPrev description is TBD
const PropertyPrev = "prev"

// PropertyPreview description is TBD
const PropertyPreview = "preview"

// PropertyProvider description is TBD
const PropertyProvider = "provider"

// PropertyReplies description is TBD
const PropertyReplies = "replies"

// PropertyResult description is TBD
const PropertyResult = "result"

// PropertyAudience description is TBD
const PropertyAudience = "audience"

// PropertyPartOf description is TBD
const PropertyPartOf = "partOf"

// PropertyTag description is TBD
const PropertyTag = "tag"

// PropertyTags description is TBD
const PropertyTags = "tags"

// PropertyTarget description is TBD
const PropertyTarget = "target"

// PropertyTo description is TBD
const PropertyTo = "to"

// PropertyType description is TBD
const PropertyType = "type"

// PropertyURL description is TBD
const PropertyURL = "url"

// PropertyHREF description is TBD
const PropertyHREF = "href"

// PropertyDescribes description is TBD
const PropertyDescribes = "describes"

// PropertyFormerType description is TBD
const PropertyFormerType = "formerType"

/*** NON-Object Properties ***/

// PropertyClosed description is TBD
const PropertyClosed = "closed" // (xsd:dateTime)

// PropertyAccuracy description is TBD
const PropertyAccuracy = "accuracy" // (xsd:float)

// PropertyAltitude description is TBD
const PropertyAltitude = "altitude" // (xsd:float)

// PropertyContent description is TBD
const PropertyContent = "content"

// PropertyContentMap description is TBD
const PropertyContentMap = "contentMap"

// PropertyName description is TBD
const PropertyName = "name"

// PropertyNameMap description is TBD
const PropertyNameMap = "nameMap"

// PropertyDownstreamDuplicates description is TBD
const PropertyDownstreamDuplicates = "downstreamDuplicates"

// PropertyDuration description is TBD
const PropertyDuration = "duration" // (xsd:duration)

// PropertyEndTime description is TBD
const PropertyEndTime = "endTime" // (xsd:dateTime)

// PropertyHeight description is TBD
const PropertyHeight = "height" // (xsd:nonNegativeInteger)

// PropertyHrefLang description is TBD
const PropertyHrefLang = "hreflang"

// PropertyLatitude description is TBD
const PropertyLatitude = "latitude" // (xsd:float)

// PropertyLongitude description is TBD
const PropertyLongitude = "longitude" // (xsd:float)

// PropertyMediaType description is TBD
const PropertyMediaType = "mediaType"

// PropertyPublished description is TBD
const PropertyPublished = "published" // (xsd:dateTime)

// PropertyRadius description is TBD
const PropertyRadius = "radius" // (xsd:float)

// PropertyRating description is TBD
const PropertyRating = "rating" // (xsd:float)

// PropertyRel description is TBD
const PropertyRel = "rel"

// PropertyStartIndex description is TBD
const PropertyStartIndex = "startIndex" // (xsd:nonNegativeInteger)

// PropertyStartTime description is TBD
const PropertyStartTime = "startTime" // (xsd:dateTime)

// PropertySummary description is TBD
const PropertySummary = "summary"

// PropertySummaryMap description is TBD
const PropertySummaryMap = "summaryMap"

// PropertyTotalItems description is TBD
const PropertyTotalItems = "totalItems" // (xsd:nonNegativeInteger)

// PropertyUnits description is TBD
const PropertyUnits = "units"

// PropertyUpdated description is TBD
const PropertyUpdated = "updated" // (xsd:dateTime)

// PropertyUpstreamDuplicates description is TBD
const PropertyUpstreamDuplicates = "upstreamDuplicates"

// PropertyVerb description is TBD
const PropertyVerb = "verb"

// PropertyWidth description is TBD
const PropertyWidth = "width" // (xsd:nonNegativeInteger)

// PropertyDeleted description is TBD
const PropertyDeleted = "deleted" // (xsd:dateTime)
