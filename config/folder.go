package config

// FolderAdapterEmbed identifies a folder that is compiled into the Emissary binary
const FolderAdapterEmbed = "EMBED"

// FolderAdapterFile identifies a folder on the local filesystem
const FolderAdapterFile = "FILE"

// FolderAdapterGit identifies a folder that is cloned from a Git repository
const FolderAdapterGit = "GIT"

// FolderAdapterS3 identifies a folder in an S3-compatible object store
const FolderAdapterS3 = "S3"
