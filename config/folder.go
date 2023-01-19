package config

const FolderAdapterEmbed = "EMBED"
const FolderAdapterFile = "FILE"
const FolderAdapterGit = "GIT"
const FolderAdapterS3 = "S3"

type Folder struct {
	Adapter  string `json:"adapter"`
	Location string `json:"location"`
}
