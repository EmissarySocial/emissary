package s3uri

import (
	"testing"
)

func TestS3URI(t *testing.T) {
	s3u := NewS3URI()
	// s3u.ParseString("s3://test123")
	// s3u.URI()
	// t.Log(s3u)
	// s3u.Bucket = String("test")
	// s3u.URI().String()
	// t.Log(s3u)

	s3u.ParseString("s3://test123/")
	// t.Log(s3u)
	s3u.ParseString("s3://test123/key456")
	// t.Log(s3u)
	s3u.ParseString("s3://test123/key456/")
	// t.Log(s3u)
	// s3u.ParseString("https://s3.amazonaws.com/test123")
	// s3u.ParseString("https://s3.amazonaws.com/test123/")
	// s3u.ParseString("https://s3.amazonaws.com/test123/key456")
	// s3u.ParseString("https://s3.amazonaws.com/test123/key456/")
	s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456/")
	// t.Log(s3u)
	s3u.ParseString("https://s3.eu-west-1.amazonaws.com/test123/key456/")
	// t.Log(s3u)
	s3u.ParseString("https://s3.dualstack.eu-west-1.amazonaws.com/test123/key456/")
	// t.Log(s3u)
	s3u.ParseString("https://test123.s3-website-eu-west-1.amazonaws.com/key456/")
	// t.Log(s3u)
	s3u.ParseString("https://test123.s3-accelerated.amazonaws.com/key456/")
	// t.Log(s3u)
	s3u.ParseString("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/")
	// t.Log(s3u)
	// s3u.ParseString("https://test123.s3.amazonaws.com/")
	// s3u.ParseString("https://test123.s3.amazonaws.com/key456")
	// s3u.ParseString("https://test123.s3.amazonaws.com/key456")
	// s3u.ParseString("https://google.com")) // invalid S3 endpoin

	// s3u.ParseString("https://test123.s3.amazonaws.com/key456?versionId=123456&x=1&y=2&y=3;z")
	// *s3u.Bucket, *s3u.Key, *s3u.Region, *s3u.PathStyle, *s3u.VersionID
	// s3u.URI().Scheme

	// s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456?t=this+is+a+simple+%26+short+test.")

	// u, _ := url.Parse("s3://test123/key456")
	// s3u.Parse(u)

	// MustParse(s3u.ParseString("s3://test123/key456"))
	// Will panic: no hostname
	// MustParse(s3u.ParseString(""))

	// s3u = NewS3URI(
	// 	WithRegion("eu-west-1"),
	// 	WithVersionID("12341234"),
	// 	WithNormalizedKey(true),
	// )
	// t.Log(s3u.URI())
	// s3u.ParseString("https://test123.s3.amazonaws.com/key456/?versionId=123456&x=1&y=2&y=3;z")
	// *s3u.Bucket, *s3u.Key, *s3u.Region, *s3u.PathStyle, *s3u.VersionID
	// s3u.URI().Scheme
	// t.Log(s3u.URI())
	Validate("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/")
	Validate("ftp://google.com/")
	ParseString("ftp://google.com/")
}
