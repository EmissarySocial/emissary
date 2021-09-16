package vocabulary

//  These types are defined by the W3C on https:// www.w3.org/TR/activitystreams-vocabulary/#object-types

/**** Object Types ****/

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
