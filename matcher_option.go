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
// Thr transformer function should be safe for concurrent use.
func WithPathTransformer(transformer func(pathname string) string) GlobOption {
	return func(o *globOptions) error {
		o.PathTransform = transformer
		return nil
	}
}
