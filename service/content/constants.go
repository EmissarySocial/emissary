package content

/******************************************
 * Size Limits
 ******************************************/

// maxContentBytes is the largest body an Adapter will read from a remote source.  It matches the
// maxLength of content.raw in the article Templates, so an Adapter can never return a body that
// the Stream schema would reject.
const maxContentBytes = 1 << 20

// maxFrontMatterBytes is the largest block of YAML front matter that will be parsed
const maxFrontMatterBytes = 64 << 10

// maxURLLength is the longest source address that will be accepted
const maxURLLength = 2048

/******************************************
 * Media Types
 ******************************************/

// mediaTypeMarkdown is the media type that a forge serves a .md file as when it knows what it is
const mediaTypeMarkdown = "text/markdown"

// mediaTypePlain is what every forge surveyed actually returns for a raw Markdown file
const mediaTypePlain = "text/plain"

/******************************************
 * Front Matter
 ******************************************/

// frontMatterDelimiter is the line that opens and closes a block of YAML front matter
const frontMatterDelimiter = "---"

// byteOrderMark is the UTF-8 byte-order mark that some editors write at the start of a file
const byteOrderMark = "\xef\xbb\xbf"
