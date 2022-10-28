package model

type Content struct {
	Format string `json:"format" bson:"format" path:"format"`
	Raw    string `json:"raw"    bson:"raw"    path:"raw"`
	HTML   string `json:"html"   bson:"html"   path:"html"`
}
