package s3uri

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestS3URI verifies that every supported S3 address style parses, and that non-S3 URLs are rejected
func TestS3URI(t *testing.T) {
	s3u := NewS3URI()

	result, err := s3u.ParseString("s3://test123/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("s3://test123/key456")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("s3://test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123/key456")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3.eu-west-1.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3.dualstack.eu-west-1.amazonaws.com/test123/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://test123.s3-website-eu-west-1.amazonaws.com/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://test123.s3-accelerated.amazonaws.com/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://google.com") // not an S3 endpoint
	require.Error(t, err)
	require.Nil(t, result)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456?versionId=123456&x=1&y=2&y=3;z")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("https://s3-eu-west-1.amazonaws.com/test123/key456?t=this+is+a+simple+%26+short+test.")
	require.Nil(t, err)
	require.NotNil(t, result)

	result, err = s3u.ParseString("s3://test123/key456")
	require.Nil(t, err)
	require.NotNil(t, result)

	// An empty string has no hostname to parse
	_, err = s3u.ParseString("")
	require.Error(t, err)

	result, err = s3u.ParseString("https://test123.s3.amazonaws.com/key456/?versionId=123456&x=1&y=2&y=3;z")
	require.Nil(t, err)
	require.NotNil(t, result)
	Validate("https://test123.s3-accelerated.dualstack.amazonaws.com/key456/")
	Validate("ftp://google.com/")

	result, err = ParseString("ftp://google.com/")
	require.Error(t, err)
	require.Nil(t, result)
}
