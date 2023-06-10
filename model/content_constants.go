package model

// ContentFormatHTML represents a content object whose Raw value is defined in HTML
// This content can be used in a browser (after passing through a safety filter like BlueMonday)
const ContentFormatHTML = "HTML"

// ContentFormatText represents a content object whose Raw value is defined in plain text.
// This content must be converted into HTML before being used in a browser
const ContentFormatText = "TEXT"

// ContentFormatContentJS represents a content object whose Raw value is defined in Markdown
// This content must be converted into HTML before being used in a browser
// See: https://commonmark.org
const ContentFormatMarkdown = "MARKDOWN"

// ContentFormatEditorJS represents a content object whose Raw value is defined in EditorJS
// This content must be converted into HTML before being used in a browser
// See: https://editorjs.io
const ContentFormatEditorJS = "EDITORJS"
