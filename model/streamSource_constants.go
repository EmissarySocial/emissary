package model

// StreamSourceMethodHTTPS identifies a StreamSource record that reads a Markdown file from a URL
const StreamSourceMethodHTTPS = "HTTPS"

// StreamSourceStatusNew marks a StreamSource record that has never been synchronized
const StreamSourceStatusNew = "NEW"

// StreamSourceStatusLoading marks a StreamSource record whose synchronization is under way
const StreamSourceStatusLoading = "LOADING"

// StreamSourceStatusSuccess marks a StreamSource record whose last synchronization succeeded
const StreamSourceStatusSuccess = "SUCCESS"

// StreamSourceStatusFailure marks a StreamSource record whose last synchronization failed
const StreamSourceStatusFailure = "FAILURE"
