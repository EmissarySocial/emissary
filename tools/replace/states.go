package replace

// stateReady is the scanning state where matches are replaced.
const stateReady = 0

// stateSkipHTML copies the inside of an HTML tag verbatim, until the closing ">".
const stateSkipHTML = 1

// stateInsideAnchor copies the text content of an <a> element verbatim, until its </a>, so that
// already-linkified tags are never replaced (and never nested) on a second pass.
const stateInsideAnchor = 2
