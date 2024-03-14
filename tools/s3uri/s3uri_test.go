package s3uri

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestS3URI(t *testing.T) {
	s3u := NewS3URI()
	// result, err := s3u.ParseString("s3://test123")
	// s3u.URI()
	// s3u.Bucket = String("test")
	// s3u.URI().String()

	result, err := s3u.ParseString("s3://test123/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("s3://test123/key456")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("s3://test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123/key456")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://s3.eu-west-1.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://s3.dualstack.eu-west-1.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://test123.s3-website-eu-west-1.amazonaws.com/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://test123.s3-accelerated.amazonaws.com/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://google.com") //) // invalid S3 endpoint
	require.Error(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456?versionId=123456&x=1&y=2&y=3;z")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)
	// *s3u.Bucket, *s3u.Key, *s3u.Region, *s3u.PathStyle, *s3u.VersionID
	// s3u.URI().Scheme

	//
	result, err = s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456?t=this+is+a+simple+%26+short+test.")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	// u, _ := url.Parse("s3://test123/key456")
	// s3u.Parse(u)

	// MustParse(
	result, err = s3u.ParseString("s3://test123/key456")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)

	// Will panic: no hostname
	// MustParse(
	_, err = s3u.ParseString("")
	require.Error(t, err)

	// s3u = NewS3URI(
	// 	WithRegion("eu-west-1"),
	// 	WithVersionID("12341234"),
	// 	WithNormalizedKey(true),
	// )
	// t.Log(s3u.URI())
	//
	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456/?versionId=123456&x=1&y=2&y=3;z")
	require.Nil(t, err)
	require.NotNil(t, result)
	// t.Log(result)
	// *s3u.Bucket, *s3u.Key, *s3u.Region, *s3u.PathStyle, *s3u.VersionID
	// s3u.URI().Scheme
	// t.Log(s3u.URI())
	Validate("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/")
	Validate("ftp://google.com/")

	result, err = ParseString("ftp://google.com/")
	require.Error(t, err)
	require.NotNil(t, result)
	// t.Log(result)
}
