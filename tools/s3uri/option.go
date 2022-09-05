package s3uri

type Option func(*S3URI)

func WithScheme(s string) Option {
	return func(s3u *S3URI) {
		s3u.Scheme = String(s)
	}
}

func WithBucket(s string) Option {
	return func(s3u *S3URI) {
		s3u.Bucket = String(s)
	}
}

func WithKey(s string) Option {
	return func(s3u *S3URI) {
		s3u.Key = String(s)
	}
}

func WithVersionID(s string) Option {
	return func(s3u *S3URI) {
		s3u.VersionID = String(s)
	}
}

func WithRegion(s string) Option {
	return func(s3u *S3URI) {
		s3u.Region = String(s)
	}
}

func WithNormalizedKey(b bool) Option {
	return func(s3u *S3URI) {
		s3u.normalize = Bool(b)
	}
}

func WithCredenials(username string, password string) Option {
	return func(s3u *S3URI) {
		s3u.AccessKey = String(username)
		s3u.Secret = String(password)
	}
}
