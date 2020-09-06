package matcher

type multiMatcher []Matcher

// Multi returns a new Matcher that matches against many matchers.
func Multi(matches ...Matcher) Matcher {
	return multiMatcher(matches)
}

// Match performs a match with all matchers provided and returns a result
// early if one matched.
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
