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

// mediaTypeXMarkdown is the pre-standard spelling of the same thing, still served by some hosts
const mediaTypeXMarkdown = "text/x-markdown"

// mediaTypePlain is what every forge surveyed actually returns for a raw file, Markdown or not
const mediaTypePlain = "text/plain"

// mediaTypeHTML is what a web server declares for a page, including a forge's file PAGE
const mediaTypeHTML = "text/html"

// mediaTypeXHTML is the XML serialization of HTML
const mediaTypeXHTML = "application/xhtml+xml"

/******************************************
 * File Extensions
 ******************************************/

// extensionMarkdown and extensionMarkdownLong name a Markdown file
const extensionMarkdown = ".md"
const extensionMarkdownLong = ".markdown"

// extensionHTML and extensionHTMLShort name an HTML file
const extensionHTML = ".html"
const extensionHTMLShort = ".htm"

/******************************************
 * Front Matter
 ******************************************/

// frontMatterDelimiter is the line that opens and closes a block of YAML front matter
const frontMatterDelimiter = "---"

// byteOrderMark is the UTF-8 byte-order mark that some editors write at the start of a file
const byteOrderMark = "\xef\xbb\xbf"
