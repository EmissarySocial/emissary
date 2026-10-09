package model

// AttachmentObjectTypeDomain represents an attachment that is owned by a Domain
const AttachmentObjectTypeDomain = "Domain"

// AttachmentObjectTypeSearchTag represents an attachment that is owned by a SearchTag
const AttachmentObjectTypeSearchTag = "SearchTag"

// AttachmentObjectTypeStream represents an attachment that is owned by a Stream
const AttachmentObjectTypeStream = "Stream"

// AttachmentObjectTypeUser represents an attachment that is owned by a User
const AttachmentObjectTypeUser = "User"

// AttachmentMediaTypeAudio represents an attachment that is audio
const AttachmentMediaTypeAudio = "audio"

// AttachmentMediaTypeDocument represents an attachment that is a document
const AttachmentMediaTypeDocument = "document"

// AttachmentMediaTypeImage represents an attachment that is an image
const AttachmentMediaTypeImage = "image"

// AttachmentMediaTypeVideo represents an attachment that is a video
const AttachmentMediaTypeVideo = "video"

// AttachmentMediaTypeAny represents an attachment that is any type
const AttachmentMediaTypeAny = "any"

// AttachmentStatusReady represents an attachment that has been transcoded
// (if necessary) and is ready to download or stream
const AttachmentStatusReady = "READY"

// AttachmentStatusWorking represents an attachment that is currently
// being processed and cannot be downloaded yet.
const AttachmentStatusWorking = "WORKING"

// AttachmentStatusFailed represents an attachment whose file could not be copied from its
// SourceURL, and will not be until the next synchronization tries again
const AttachmentStatusFailed = "FAILED"

// AttachmentCategoryStreamSource is the category of every file that a StreamSource copies into
// a Stream.  Synchronization manages these, and never touches an attachment in any other category.
const AttachmentCategoryStreamSource = "stream-source"
