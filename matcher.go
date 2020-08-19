package matcher

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/saracen/walker"
)

type Result int

const (
	separator = "/"
	globstar  = "**"
)

const (
	NotMatched Result = iota
	Matched
	Follow
)

// Matcher is an interface used for matching a path against a pattern.
type Matcher interface {
	Match(pathname string) (Result, error)
}

type matcher struct {
	pattern []string
	options matchOptions
}

// NewMatcher returns a new Matcher.
//
// The Matcher returned uses the same rules as Match, but returns a result of
// either NotMatched, Matched or Follow.
//
// Follow hints to the caller that whilst the pattern wasn't matched, path
// traversal might yield matches. This allows for more efficient globbing,
// preventing path traversal where a match is impossible.
func NewMatcher(pattern string, opts ...MatchOption) Matcher {
	matcher := matcher{pattern: strings.Split(pattern, separator)}
	for _, o := range opts {
		o(&matcher.options)
	}

	if matcher.options.MatchFn == nil {
		matcher.options.MatchFn = path.Match
	}

	return matcher
}

// Match has similar behaviour to path.Match, but supports globstar.
//
// The pattern term '**' in a path portion matches zero or more subdirectories.
//
// The only possible returned error is ErrBadPattern, when the pattern
// is malformed.
func Match(pattern, pathname string, opts ...MatchOption) (bool, error) {
	result, err := NewMatcher(pattern, opts...).Match(pathname)

	return result == Matched, err
}

func (p matcher) Match(pathname string) (Result, error) {
	return match(p.pattern, strings.Split(pathname, separator), p.options.MatchFn)
}

func match(pattern, parts []string, matchFn func(pattern, name string) (matched bool, err error)) (Result, error) {
	for {
		switch {
		case len(pattern) == 0 && len(parts) == 0:
			return Matched, nil

		case len(parts) == 0:
			return Follow, nil

		case len(pattern) == 0:
			return NotMatched, nil

		case pattern[0] == globstar && len(pattern) == 1:
			return Matched, nil

		case pattern[0] == globstar:
			for i := range parts {
				result, err := match(pattern[1:], parts[i:], matchFn)
				if result == Matched || err != nil {
					return result, err
				}
			}
			return Follow, nil
		}

		matched, err := matchFn(pattern[0], parts[0])
		switch {
		case err != nil:
			return NotMatched, err

		case !matched && len(parts) == 1 && parts[0] == "":
			return Follow, nil

		case !matched:
			return NotMatched, nil
		}

		pattern = pattern[1:]
		parts = parts[1:]
	}
}

// Glob returns the pathnames and their associated os.FileInfos of all files
// matching with the Matcher provided.
//
// Patterns are matched against the path relative to the directory provided
// and path seperators are converted to '/'. Be aware that the matching
// performed by this library's Matchers are case sensitive (even on
// case-insensitive filesystems). Use WithPathTransformer(strings.ToLower)
// and NewMatcher(strings.ToLower(pattern)) to perform case-insensitive
// matching.
//
// Glob ignores any permission and I/O errors.
func Glob(ctx context.Context, dir string, matcher Matcher, opts ...GlobOption) (map[string]os.FileInfo, error) {
	var options globOptions
	for _, o := range opts {
		err := o(&options)
		if err != nil {
			return nil, err
		}
	}

	matches := make(map[string]os.FileInfo)

	var m sync.Mutex

	ignoreErrors := walker.WithErrorCallback(func(pathname string, err error) error {
		return nil
	})

	walkFn := func(pathname string, fi os.FileInfo) error {
		rel := strings.TrimPrefix(pathname, dir)
		rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
		if rel == "" {
			return nil
		}

		if fi.IsDir() {
			rel += "/"
		}

		if options.PathTransform != nil {
			rel = options.PathTransform(rel)
		}

		result, err := matcher.Match(rel)
		if err != nil {
			return err
		}

		if result == Matched {
			m.Lock()
			defer m.Unlock()

			matches[pathname] = fi
		}

		follow := result == Matched || result == Follow
		if fi.IsDir() && !follow {
			return filepath.SkipDir
		}

		return nil
	}

	return matches, walker.WalkWithContext(ctx, dir, walkFn, ignoreErrors)
}
