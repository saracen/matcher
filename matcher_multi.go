package matcher

type multiMatcher []Matcher

// NewMultiMatcher returns a new Matcher that matches against many matchers.
func NewMultiMatcher(matches ...Matcher) Matcher {
	return multiMatcher(matches)
}

// Match uses both the include and exclude patterns to determine matches.
func (p multiMatcher) Match(pathname string) (Result, error) {
	var follow bool

	for _, include := range p {
		result, err := include.Match(pathname)

		switch {
		case err != nil:
			return NotMatched, err

		case result == Matched:
			return Matched, nil

		case result == Follow:
			follow = true
		}
	}

	if follow {
		return Follow, nil
	}

	return NotMatched, nil
}
