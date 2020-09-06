# matcher

`matcher` is similar to `path.Match`, but:

- Supports globstar/doublestar (`**`).
- Provides a fast `Glob` function.
- Supports combining matchers.

## Examples

### Match

```golang
package main

import "github.com/saracen/matcher"

func main() {
    matched, err := matcher.Match("hello/**/world", "hello/foo/bar/world")
    if err != nil {
        panic(err)
    }

    if matched {
        // do something
    }
}
```

### Glob

```golang
package main

import "github.com/saracen/matcher"

func main() {
    matches, err := matcher.Glob(context.Background(), ".", matcher.New("**/*.go"))
    if err != nil {
        panic(err)
    }

    // do something with the matches
    _ = matches
}
```

### Glob with multiple patterns

```golang
package main

import "github.com/saracen/matcher"

func main() {
    matcher := matcher.Multi(
        matcher.New("**/*.go"),
        matcher.New("**/*.txt"))

    matches, err := matcher.Glob(context.Background(), ".", matcher)
    if err != nil {
        panic(err)
    }

    // do something with the matches
    _ = matches
}
```
