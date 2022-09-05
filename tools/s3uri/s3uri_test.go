package s3uri

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestS3URI(t *testing.T) {
	s3u := NewS3URI()
	//	fmt.Println(s3u.ParseString("s3://test123"))
	//	fmt.Println(s3u.URI())
	//	spew.Dump(s3u)
	//	s3u.Bucket = String("test")
	//	fmt.Println(s3u.URI().String())
	//	spew.Dump(s3u)

	fmt.Println(s3u.ParseString("s3://test123/"))
	spew.Dump(s3u)
	fmt.Println(s3u.ParseString("s3://test123/key456"))
	spew.Dump(s3u)
	fmt.Println(s3u.ParseString("s3://test123/key456/"))
	spew.Dump(s3u)
	//	fmt.Println(s3u.ParseString("https://s3.amazonaws.com/test123"))
	//	fmt.Println(s3u.ParseString("https://s3.amazonaws.com/test123/"))
	//	fmt.Println(s3u.ParseString("https://s3.amazonaws.com/test123/key456"))
	//	fmt.Println(s3u.ParseString("https://s3.amazonaws.com/test123/key456/"))
	fmt.Println(s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456/"))
	spew.Dump(s3u)
	fmt.Println(s3u.ParseString("https://s3.eu-west-1.amazonaws.com/test123/key456/"))
	spew.Dump(s3u)
	fmt.Println(s3u.ParseString("https://s3.dualstack.eu-west-1.amazonaws.com/test123/key456/"))
	spew.Dump(s3u)
	fmt.Println(s3u.ParseString("https://test123.s3-website-eu-west-1.amazonaws.com/key456/"))
	spew.Dump(s3u)
	fmt.Println(s3u.ParseString("https://test123.s3-accelerated.amazonaws.com/key456/"))
	spew.Dump(s3u)
	fmt.Println(s3u.ParseString("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/"))
	spew.Dump(s3u)
	//	fmt.Println(s3u.ParseString("https://test123.s3.amazonaws.com/"))
	//	fmt.Println(s3u.ParseString("https://test123.s3.amazonaws.com/key456"))
	//	fmt.Println(s3u.ParseString("https://test123.s3.amazonaws.com/key456"))
	//	fmt.Println(s3u.ParseString("https://google.com")) // invalid S3 endpoint

	//	fmt.Println(s3u.ParseString("https://test123.s3.amazonaws.com/key456?versionId=123456&x=1&y=2&y=3;z"))
	//	fmt.Println(*s3u.Bucket, *s3u.Key, *s3u.Region, *s3u.PathStyle, *s3u.VersionID)
	//	fmt.Println(s3u.URI().Scheme)

	//	fmt.Println(s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456?t=this+is+a+simple+%26+short+test."))

	//	u, _ := url.Parse("s3://test123/key456")
	//	fmt.Println(s3u.Parse(u))

	//	fmt.Println(MustParse(s3u.ParseString("s3://test123/key456")))
	//	// Will panic: no hostname
	//	// fmt.Println(MustParse(s3u.ParseString("")))

	//	s3u = NewS3URI(
	//		WithRegion("eu-west-1"),
	//		WithVersionID("12341234"),
	//		WithNormalizedKey(true),
	//	)
	//	spew.Dump(s3u.URI())
	//	fmt.Println(s3u.ParseString("https://test123.s3.amazonaws.com/key456/?versionId=123456&x=1&y=2&y=3;z"))
	//	fmt.Println(*s3u.Bucket, *s3u.Key, *s3u.Region, *s3u.PathStyle, *s3u.VersionID)
	//	fmt.Println(s3u.URI().Scheme)
	//	spew.Dump(s3u.URI())
	fmt.Println(Validate("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/"))
	fmt.Println(Validate("ftp://google.com/"))
	fmt.Println(ParseString("ftp://google.com/"))
}
