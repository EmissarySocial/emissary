package s3uri

// Option is a configuration function that modifies an S3URI
type Option func(*S3URI)

// WithScheme sets the URI scheme (e.g. "s3" or "https")
func WithScheme(s string) Option {
	return func(s3u *S3URI) {
		s3u.Scheme = String(s)
	}
}

// WithBucket sets the S3 bucket name
func WithBucket(s string) Option {
	return func(s3u *S3URI) {
		s3u.Bucket = String(s)
	}
}

// WithKey sets the S3 object key
func WithKey(s string) Option {
	return func(s3u *S3URI) {
		s3u.Key = String(s)
	}
}

// WithVersionID sets the S3 object version
func WithVersionID(s string) Option {
	return func(s3u *S3URI) {
		s3u.VersionID = String(s)
	}
}

// WithRegion sets the AWS region
func WithRegion(s string) Option {
	return func(s3u *S3URI) {
		s3u.Region = String(s)
	}
}

// WithNormalizedKey controls whether the object key is normalized when the URI is generated
func WithNormalizedKey(b bool) Option {
	return func(s3u *S3URI) {
		s3u.normalize = Bool(b)
	}
}

// WithCredenials sets the access key and secret used to sign requests
func WithCredenials(username string, password string) Option {
	return func(s3u *S3URI) {
		s3u.AccessKey = String(username)
		s3u.Secret = String(password)
	}
}
