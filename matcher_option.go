package matcher

// GlobOption is an option to configure Glob() behaviour.
type GlobOption func(*globOptions) error

type globOptions struct {
	PathTransform func(string) string
}

// WithPathTransforms allows a function to transform a path prior to it being
// matched. A common use-case is WithPathTransformer(strings.ToLower) to ensure
// paths have their case folded before matching.
//
// The transformer function should be safe for concurrent use.
func WithPathTransformer(transformer func(pathname string) string) GlobOption {
	return func(o *globOptions) error {
		o.PathTransform = transformer
		return nil
	}
}

// MatchOption is an option to configure Match() behaviour.
type MatchOption func(*matchOptions)

type matchOptions struct {
	MatchFn func(pattern, name string) (matched bool, err error)
}

// WithMatchFunc allows a user provided matcher to be used in place of
// path.Match for matching path segments. The globstar pattern will always be
// supported, but paths between directory separators will be matched against
// the function provided.
func WithMatchFunc(matcher func(pattern, name string) (matched bool, err error)) MatchOption {
	return func(o *matchOptions) {
		o.MatchFn = matcher
	}
}
