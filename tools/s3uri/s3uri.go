// Package s3uri parses and generates URI strings for Amazon S3 resources.
// it is based on this original Gist by https://github.com/kwilczynski
// https://gist.github.com/kwilczynski/f6e626990d6d2395b42a12721b165b86
package s3uri

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// DefaultRegion contains a default region for an S3 bucket, when a region
// cannot be determined, for example when the s3:// schema is used or when
// path style URL has been given without the region component in the
// fully-qualified domain name.
const DefaultRegion = "us-east-1"

var (
	ErrBucketNotFound    = errors.New("bucket name could not be found")
	ErrHostnameNotFound  = errors.New("hostname could not be found")
	ErrInvalidS3Endpoint = errors.New("an invalid S3 endpoint URL")

	// Pattern used to parse multiple path and host style S3 endpoint URLs.
	s3URLPattern = regexp.MustCompile(`^(.+\.)?s3[.-](?:(accelerated|dualstack|website)[.-])?([a-z0-9-]+)\.`)
)

type S3URI struct {
	uri       *url.URL
	options   []Option
	normalize *bool

	HostStyle   *bool
	PathStyle   *bool
	Accelerated *bool
	DualStack   *bool
	Website     *bool

	Scheme    *string
	Bucket    *string
	Key       *string
	VersionID *string
	Region    *string

	AccessKey *string
	Secret    *string
}

func NewS3URI(opts ...Option) *S3URI {
	return &S3URI{options: opts}
}

func (s3u *S3URI) Reset() *S3URI {
	return reset(s3u)
}

func (s3u *S3URI) Parse(v any) (*S3URI, error) {
	return parse(s3u, v)
}

func (s3u *S3URI) ParseURL(u *url.URL) (*S3URI, error) {
	return parse(s3u, u)
}

func (s3u *S3URI) ParseString(s string) (*S3URI, error) {
	return parse(s3u, s)
}

func (s3u *S3URI) URI() *url.URL {
	return s3u.uri
}

func (s3u *S3URI) HasCredentials() bool {
	return (s3u.AccessKey != nil) && (s3u.Secret != nil)
}

func (s3u *S3URI) GetCredentials() (string, string, string) {
	return *s3u.AccessKey, *s3u.Secret, ""
}

func Parse(v any) (*S3URI, error) {
	return NewS3URI().Parse(v)
}

func ParseURL(u *url.URL) (*S3URI, error) {
	return NewS3URI().ParseURL(u)
}

func ParseString(s string) (*S3URI, error) {
	return NewS3URI().ParseString(s)
}

func MustParse(s3u *S3URI, err error) *S3URI {
	if err != nil {
		panic(err)
	}
	return s3u
}

func Validate(v any) bool {
	_, err := NewS3URI().Parse(v)
	return err == nil
}

func ValidateURL(u *url.URL) bool {
	_, err := NewS3URI().Parse(u)
	return err == nil
}

func ValidateString(s string) bool {
	_, err := NewS3URI().Parse(s)
	return err == nil
}

func parse(s3u *S3URI, s any) (*S3URI, error) {
	var (
		u   *url.URL
		err error
	)

	switch s := s.(type) {
	case string:
		u, err = url.Parse(s)
	case *url.URL:
		u = s
	default:
		return nil, fmt.Errorf("unable to parse unknown type: %T", s)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to parse given S3 endpoint URL: %w", err)
	}

	reset(s3u)
	s3u.uri = u

	switch u.Scheme {
	case "s3", "http", "https":
		s3u.Scheme = String(u.Scheme)
	default:
		return nil, fmt.Errorf("unable to parse schema type: %s", u.Scheme)
	}

	// Handle S3 endpoint URL with the schema s3:// that is neither
	// the host style nor the path style.
	if u.Scheme == "s3" {
		if u.Host == "" {
			return nil, ErrBucketNotFound
		}
		s3u.Bucket = String(u.Host)

		if u.Path != "" && u.Path != "/" {
			s3u.Key = String(u.Path[1:len(u.Path)])
		}
		s3u.Region = String(DefaultRegion)

		return s3u, nil
	}

	if u.Host == "" {
		return nil, ErrHostnameNotFound
	}

	if u.User != nil {
		s3u.AccessKey = String(u.User.Username())
		if password, ok := u.User.Password(); ok {
			s3u.Secret = String(password)
		}
	}

	matches := s3URLPattern.FindStringSubmatch(u.Host)
	if len(matches) < 1 {
		return nil, ErrInvalidS3Endpoint
	}

	prefix := matches[1]
	usage := matches[2] // Type of the S3 bucket.
	region := matches[3]

	if prefix == "" {
		s3u.PathStyle = Bool(true)

		if u.Path != "" && u.Path != "/" {
			u.Path = u.Path[1:len(u.Path)]

			switch index := strings.Index(u.Path, "/"); {

			case index == -1:
				s3u.Bucket = String(u.Path)

			case index == len(u.Path)-1:
				s3u.Bucket = String(u.Path[:index])

			default:
				s3u.Bucket = String(u.Path[:index])
				s3u.Key = String(u.Path[index+1:])
			}
		}
	} else {
		s3u.HostStyle = Bool(true)
		s3u.Bucket = String(prefix[:len(prefix)-1])

		if u.Path != "" && u.Path != "/" {
			s3u.Key = String(u.Path[1:len(u.Path)])
		}
	}

	const (
		// Used to denote type of the S3 bucket.
		accelerated = "accelerated"
		dualStack   = "dualstack"
		website     = "website"

		// Part of the amazonaws.com domain name.  Set when no region
		// could be ascertain correctly using the S3 endpoint URL.
		amazonAWS = "amazonaws"

		// Part of the query parameters.  Used when retrieving S3
		// object (key) of a particular version.
		versionID = "versionId"
	)

	// An S3 bucket can be either accelerated or website endpoint,
	// but not both.
	switch usage {
	case accelerated:
		s3u.Accelerated = Bool(true)
	case website:
		s3u.Website = Bool(true)
	}

	// An accelerated S3 bucket can also be dualstack.
	if usage == dualStack || region == dualStack {
		s3u.DualStack = Bool(true)
	}

	// Handle the special case of an accelerated dualstack S3
	// endpoint URL:
	//   <BUCKET>.s3-accelerated.dualstack.amazonaws.com/<KEY>.
	// As there is no way to accertain the region solely based on
	// the S3 endpoint URL.
	if usage != accelerated {
		s3u.Region = String(DefaultRegion)
		if region != amazonAWS {
			s3u.Region = String(region)
		}
	}

	// Query string used when requesting a particular version of a given
	// S3 object (key).
	if s := u.Query().Get(versionID); s != "" {
		s3u.VersionID = String(s)
	}

	// Apply options that serve as overrides after the initial parsing
	// is completed.  This allows for bucket name, key, version ID, etc.,
	// to be overridden at the parsing stage.
	for _, o := range s3u.options {
		o(s3u)
	}

	// Remove trailing slash from the key name, so that the "key/" will
	// become "key" and similarly "a/complex/key/" will simply become
	// "a/complex/key" afer being normalized.
	if BoolValue(s3u.normalize) && s3u.Key != nil {
		k := StringValue(s3u.Key)
		if k[len(k)-1] == '/' {
			k = k[:len(k)-1]
		}
		s3u.Key = String(k)
	}

	return s3u, nil
}

// Reset fields in the S3URI type, and set boolean values to false.
func reset(s3u *S3URI) *S3URI {
	*s3u = S3URI{
		HostStyle:   Bool(false),
		PathStyle:   Bool(false),
		Accelerated: Bool(false),
		DualStack:   Bool(false),
		Website:     Bool(false),
	}
	return s3u
}

func String(s string) *string {
	return &s
}

func Bool(b bool) *bool {
	return &b
}

func StringValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func BoolValue(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}
