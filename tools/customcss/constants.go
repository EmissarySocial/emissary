package customcss

// SchemaFormat names the rosetta string format that accepts only a stylesheet that Validate accepts
const SchemaFormat = "custom-css"

// SchemaArgumentRemote lets the format accept files from other websites, written `custom-css=remote`
const SchemaArgumentRemote = "remote"

// maxLength is the longest stylesheet, in bytes, that Validate will read
const maxLength = 64 * 1024

// cssWhitespace lists the characters that CSS treats as whitespace between tokens
const cssWhitespace = " \t\r\n\f"
