package config

const FolderAdapterEmbed = "EMBED"
const FolderAdapterFile = "FILE"
const FolderAdapterGit = "GIT"
const FolderAdapterHTTP = "HTTP"
const FolderAdapterS3 = "S3"

type Folder struct {
	Adapter  string `path:"adapter"  json:"adapter"`
	Location string `path:"location" json:"location"`
}
