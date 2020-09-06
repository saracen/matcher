package matcher

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	// "github.com/bmatcuk/doublestar"
	// "github.com/saracen/walker"
)

var ErrBadPattern = path.ErrBadPattern

type MatchTest struct {
	pattern, s string
	result     Result
	err        error
}

var matchTests = map[string][]MatchTest{
	"path.Match": {
		// https://golang.org/src/path/match_test.go
		{"abc", "abc", Matched, nil},
		{"*", "abc", Matched, nil},
		{"*c", "abc", Matched, nil},
		{"a*", "a", Matched, nil},
		{"a*", "abc", Matched, nil},
		{"a*", "ab/c", NotMatched, nil},
		{"a*/b", "abc/b", Matched, nil},
		{"a*/b", "a/c/b", NotMatched, nil},
		{"a*b*c*d*e*/f", "axbxcxdxe/f", Matched, nil},
		{"a*b*c*d*e*/f", "axbxcxdxexxx/f", Matched, nil},
		{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", NotMatched, nil},
		{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", NotMatched, nil},
		{"a*b?c*x", "abxbbxdbxebxczzx", Matched, nil},
		{"a*b?c*x", "abxbbxdbxebxczzy", NotMatched, nil},
		{"ab[c]", "abc", Matched, nil},
		{"ab[b-d]", "abc", Matched, nil},
		{"ab[e-g]", "abc", NotMatched, nil},
		{"ab[^c]", "abc", NotMatched, nil},
		{"ab[^b-d]", "abc", NotMatched, nil},
		{"ab[^e-g]", "abc", Matched, nil},
		{"a\\*b", "a*b", Matched, nil},
		{"a\\*b", "ab", NotMatched, nil},
		{"a?b", "a☺b", Matched, nil},
		{"a[^a]b", "a☺b", Matched, nil},
		{"a???b", "a☺b", NotMatched, nil},
		{"a[^a][^a][^a]b", "a☺b", NotMatched, nil},
		{"[a-ζ]*", "α", Matched, nil},
		{"*[a-ζ]", "A", NotMatched, nil},
		{"a?b", "a/b", NotMatched, nil},
		{"a*b", "a/b", NotMatched, nil},
		{"[\\]a]", "]", Matched, nil},
		{"[\\-]", "-", Matched, nil},
		{"[x\\-]", "x", Matched, nil},
		{"[x\\-]", "-", Matched, nil},
		{"[x\\-]", "z", NotMatched, nil},
		{"[\\-x]", "x", Matched, nil},
		{"[\\-x]", "-", Matched, nil},
		{"[\\-x]", "a", NotMatched, nil},
		{"[]a]", "]", NotMatched, ErrBadPattern},
		{"[-]", "-", NotMatched, ErrBadPattern},
		{"[x-]", "x", NotMatched, ErrBadPattern},
		{"[x-]", "-", NotMatched, ErrBadPattern},
		{"[x-]", "z", NotMatched, ErrBadPattern},
		{"[-x]", "x", NotMatched, ErrBadPattern},
		{"[-x]", "-", NotMatched, ErrBadPattern},
		{"[-x]", "a", NotMatched, ErrBadPattern},
		{"\\", "a", NotMatched, ErrBadPattern},
		{"[a-b-c]", "a", NotMatched, ErrBadPattern},
		{"[", "a", NotMatched, ErrBadPattern},
		{"[^", "a", NotMatched, ErrBadPattern},
		{"[^bc", "a", NotMatched, ErrBadPattern},
		{"a[", "a", NotMatched, nil},
		{"a[", "ab", NotMatched, ErrBadPattern},
		{"*x", "xxx", Matched, nil},
	},
	"t3070-wildmatch basic wildmatch features": {
		{"foo", "foo", Matched, nil},
		{"bar", "foo", NotMatched, nil},
		{"", "", Matched, nil},
		{"???", "foo", Matched, nil},
		{"??", "foo", NotMatched, nil},
		{"*", "foo", Matched, nil},
		{"f*", "foo", Matched, nil},
		{"*f", "foo", NotMatched, nil},
		{"*foo*", "foo", Matched, nil},
		{"*ob*a*r*", "foobar", Matched, nil},
		{"*ab", "aaaaaaabababab", Matched, nil},
		{`foo\*`, "foo*", Matched, nil},
		{`foo\*bar`, "foobar", NotMatched, nil},
		{`f\\oo`, `f\oo`, Matched, nil},
		{"*[al]?", "ball", Matched, nil},
		{"[ten]", "ten", NotMatched, nil},
		{"**[^te]", "ten", Matched, nil},
		{"**[^ten]", "ten", NotMatched, nil},
		{"t[a-g]n", "ten", Matched, nil},
		{"t[^a-g]n", "ten", NotMatched, nil},
		{"t[^a-g]n", "ton", Matched, nil},
		{`a[\]]b`, "a]b", Matched, nil},
		{`a[\]\-]b`, "a-b", Matched, nil},
		{`a[\]\-]b`, "a]b", Matched, nil},
		{`a[\]\-]b`, "aab", NotMatched, nil},
		{`a[\]a\-]b`, "aab", Matched, nil},
		{"]", "]", Matched, nil},
	},
	"t3070-wildmatch extended slash-matching features": {
		{"foo*bar", "foo/baz/bar", NotMatched, nil},
		{"foo**bar", "foo/baz/bar", NotMatched, nil},
		{"foo**bar", "foobazbar", Matched, nil},
		{"foo/**/bar", "foo/baz/bar", Matched, nil},
		{"foo/**/**/bar", "foo/baz/bar", Matched, nil},
		{"foo/**/bar", "foo/b/a/z/bar", Matched, nil},
		{"foo/**/**/bar", "foo/b/a/z/bar", Matched, nil},
		{"foo/**/bar", "foo/bar", Matched, nil},
		{"foo/**/**/bar", "foo/bar", Matched, nil},
		{"foo?bar", "foo/bar", NotMatched, nil},
		{"foo[/]bar", "foo/bar", NotMatched, nil},
		{"foo[^a-z]bar", "foo/bar", NotMatched, nil},
		{"f[^eiu][^eiu][^eiu][^eiu][^eiu]r", "foo/bar", NotMatched, nil},
		{"f[^eiu][^eiu][^eiu][^eiu][^eiu]r", "foo-bar", Matched, nil},
		{"**/foo", "foo", Matched, nil},
		{"**/foo", "XXX/foo", Matched, nil},
		{"**/foo", "bar/baz/foo", Matched, nil},
		{"*/foo", "bar/baz/foo", NotMatched, nil},
		{"**/bar*", "foo/bar/baz", Follow, nil},
		{"**/bar/*", "deep/foo/bar/baz", Matched, nil},
		{"**/bar/*", "deep/foo/bar/baz/", Follow, nil},
		{"**/bar/**", "deep/foo/bar/baz/", Matched, nil},
		{"**/bar/*", "deep/foo/bar", Follow, nil},
		{"**/bar/**", "deep/foo/bar/", Matched, nil},
		{"**/bar**", "foo/bar/baz", Follow, nil},
		{"*/bar/**", "foo/bar/baz/x", Matched, nil},
		{"*/bar/**", "deep/foo/bar/baz/x", NotMatched, nil},
		{"**/bar/*/*", "deep/foo/bar/baz/x", Matched, nil},
	},
	"t3070-wildmatch various additional tests": {
		{"a[c-c]st", "acrt", NotMatched, nil},
		{"a[c-c]rt", "acrt", Matched, nil},
		{"[!]-]", "]", NotMatched, nil},
		{"[!]-]", "a", NotMatched, nil},
		{`\`, "", Follow, nil},
		{`\`, `\`, NotMatched, ErrBadPattern},
		{`*/\`, `XXX/\`, NotMatched, ErrBadPattern},
		{`*/\\`, `XXX/\`, Matched, nil},
		{"foo", "foo", Matched, nil},
		{"@foo", "@foo", Matched, nil},
		{"@foo", "foo", NotMatched, nil},
		{`\[ab]`, "[ab]", Matched, nil},
		{"[[]ab]", "[ab]", Matched, nil},
		{"[[:]ab]", "[ab]", Matched, nil},
		{`[\[:]ab]`, "[ab]", Matched, nil},
		{`\??\?b`, "?a?b", Matched, nil},
		{`\a\b\c`, "abc", Matched, nil},
		{"", "foo", NotMatched, nil},
		{"**/t[o]", "foo/bar/baz/to", Matched, nil},
	},
	"t3070-wildmatch additional tests, including malformed wildmatch patterns": {
		{`[\\-^]`, "]", Matched, nil},
		{`[\\-^]`, "[", NotMatched, nil},
		{`[\-_]`, "-", Matched, nil},
		{`[\]]`, "]", Matched, nil},
		{`[\]]`, `\]`, NotMatched, nil},
		{`[\]]`, `\`, NotMatched, nil},
		{"a[]b", "ab", NotMatched, ErrBadPattern},
		{"a[]b", "a[]b", NotMatched, ErrBadPattern},
		{"ab[", "ab[", NotMatched, ErrBadPattern},
		{"[^", "ab", NotMatched, ErrBadPattern},
		{"[-", "ab", NotMatched, ErrBadPattern},
		{`[\-]`, "-", Matched, nil},
		{"[a-", "-", NotMatched, ErrBadPattern},
		{"[!a-", "-", NotMatched, ErrBadPattern},
		{`[\--A]`, "-", Matched, nil},
		{`[\--A]`, "5", Matched, nil},
		{`[ -\-]`, " ", Matched, nil},
		{`[ -\-]`, "$", Matched, nil},
		{`[ -\-]`, "-", Matched, nil},
		{`[ -\-]`, "0", NotMatched, nil},
		{`[\--\-]`, "-", Matched, nil},
		{`[\--\-\--\-]`, "-", Matched, nil},
		{`[a-e\-n]`, "j", NotMatched, nil},
		{`[a-e\-n]`, "-", Matched, nil},
		{`[^\--\-\--\-]`, "a", Matched, nil},
		{`[\]-a]`, "[", NotMatched, nil},
		{`[\]-a]`, "^", Matched, nil},
		{`[^\]-a]`, "^", NotMatched, nil},
		{`[^\]-a]`, "[", Matched, nil},
		{"[a^bc]", "^", Matched, nil},
		{`[a\-]b]`, "-b]", Matched, nil},
		{`[\]`, `\`, NotMatched, ErrBadPattern},
		{`[\\]`, `\`, Matched, nil},
		{`[^\\]`, `\`, NotMatched, nil},
		{`[A-\\]`, "G", Matched, nil},
		{"b*a", "aaabbb", NotMatched, nil},
		{"*ba*", "aabcaa", NotMatched, nil},
		{"[,]", ",", Matched, nil},
		{`[\\,]`, ",", Matched, nil},
		{`[\\,]`, `\`, Matched, nil},
		{"[,-.]", "-", Matched, nil},
		{"[,-.]", "+", NotMatched, nil},
		{"[,-.]", "-.]", NotMatched, nil},
		{`[\1-\3]`, "2", Matched, nil},
		{`[\1-\3]`, "3", Matched, nil},
		{`[\1-\3]`, "4", NotMatched, nil},
		{`[[-\]]`, `\`, Matched, nil},
		{`[[-\]]`, "[", Matched, nil},
		{`[[-\]]`, "]", Matched, nil},
		{`[[-\]]`, "-", NotMatched, nil},
	},
	"t3070-wildmatch test recursion": {
		{"-*-*-*-*-*-*-12-*-*-*-m-*-*-*", "-adobe-courier-bold-o-normal--12-120-75-75-m-70-iso8859-1", Matched, nil},
		{"-*-*-*-*-*-*-12-*-*-*-m-*-*-*", "-adobe-courier-bold-o-normal--12-120-75-75-X-70-iso8859-1", NotMatched, nil},
		{"-*-*-*-*-*-*-12-*-*-*-m-*-*-*", "-adobe-courier-bold-o-normal--12-120-75-75-/-70-iso8859-1", NotMatched, nil},
		{`XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*`, `XXX/adobe/courier/bold/o/normal//12/120/75/75/m/70/iso8859/1`, Matched, nil},
		{`XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*`, `XXX/adobe/courier/bold/o/normal//12/120/75/75/X/70/iso8859/1`, NotMatched, nil},
		{`**/*a*b*g*n*t`, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txt", Matched, nil},
		{`**/*a*b*g*n*t`, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txtz", Follow, nil},
		{`*/*/*`, "foo", Follow, nil},
		{`*/*/*`, "foo/bar", Follow, nil},
		{`*/*/*`, "foo/bba/arr", Matched, nil},
		{`*/*/*`, "foo/bb/aa/rr", NotMatched, nil},
		{`**/**/**`, "foo/bb/aa/rr", Matched, nil},
		{"*X*i", "abcXdefXghi", Matched, nil},
		{"*X*i", "ab/cXd/efXg/hi", NotMatched, nil},
		{`*/*X*/*/*i`, "ab/cXd/efXg/hi", Matched, nil},
		{`**/*X*/**/*i`, "ab/cXd/efXg/hi", Matched, nil},
	},
	"follow tests": {
		{"**/test", "hello/world", Follow, nil},
		{"**/abc/**", "hello/world/abc", Follow, nil},
		{"abc/def/**/xyz", "abc", Follow, nil},
		{"abc/def/**/xyz", "abc/def", Follow, nil},
		{"abc/def/**/xyz", "abc/def/hello", Follow, nil},
		{"abc/def/**/xyz", "abc/def/hello/world", Follow, nil},
		{"abc/def/**/xyz", "abc/def/hello/world/xyz", Matched, nil},
		{"**/*", "hello/world", Matched, nil},
		{"**/abc/**", "hello/world/abc/", Matched, nil},
		{"**/**/**", "hello", Matched, nil},
		{"**/hello/world", "hello/world", Matched, nil},
		{"abc/**/hello/world", "abc/hello/world", Matched, nil},
		{"abc/**/hello/world", "xyz/abc/hello/world", NotMatched, nil},
		{"files/dir1/file1.txt", "files/", Follow, nil},
		{"files/dir1/file1.txt", "files/dir1/", Follow, nil},
		{"files/dir1/file1.txt", "files/dir1/file1.txt", Matched, nil},
	},
	"various tests": {
		{"**/doc", "value/volcano/tail/doc", Matched, nil},
		{"**/*lue/vol?ano/ta?l", "value/volcano/tail", Matched, nil},
		{"**/*lue/vol?ano/tail", "head/value/volcano/tail", Matched, nil},
		{"**/*lue/vol?ano/tail", "head/value/Volcano/tail", Follow, nil},
		{"*lue/vol?ano/**", "value/volcano/tail/moretail", Matched, nil},
		{"*lue/**", "value/volcano", Matched, nil},
		{"*lue/vol?ano/**", "value/volcano", Follow, nil},
		{"*lue/**/vol?ano", "value/volcano", Matched, nil},
		{"*lue/**/vol?ano", "value/middle/volcano", Matched, nil},
		{"*lue/**/vol?ano", "value/middle1/middle2/volcano", Matched, nil},
		{"*lue/**foo/vol?ano/tail", "value/foo/volcano/tail", Matched, nil},
		{"**/head/v[ou]l[kc]ano", "value/head/volcano", Matched, nil},
		{"**/head/v[ou]l[", "value/head/volcano", NotMatched, ErrBadPattern},
		{"**/head/v[ou]l[", "value/head/vol[", NotMatched, ErrBadPattern},
		{"value/**/v[ou]l[", "value/head/vol[", NotMatched, ErrBadPattern},
		{"**/android/**/GeneratedPluginRegistrant.java", "packages/flutter_tools/lib/src/android/gradle.dart", Follow, nil},
		{"**/*/unicorns/*.bin", "data/rainbows/unicorns/0ee357d9-bc00-4c78-8738-7debdf909d26.bin", Matched, nil},
		{"**/unicorns/*.bin", "data/rainbows/unicorns/0ee357d9-bc00-4c78-8738-7debdf909d26.bin", Matched, nil},
	},
}

func TestMatch(t *testing.T) {
	for tn, tests := range matchTests {
		tests := tests
		t.Run(tn, func(t *testing.T) {
			for _, tt := range tests {
				matched, err := Match(tt.pattern, tt.s)

				if matched && tt.result != Matched || err != tt.err {
					t.Errorf("Match(%#q, %#q) = (%v, %v) want (%v, %v)", tt.pattern, tt.s, matched, err, tt.result == Matched, tt.err)
					return
				}
			}
		})
	}
}

func TestNewMatcher(t *testing.T) {
	for tn, tests := range matchTests {
		tests := tests
		t.Run(tn, func(t *testing.T) {
			for _, tt := range tests {
				result, err := New(tt.pattern).Match(tt.s)
				if result != tt.result || err != tt.err {
					t.Errorf("New(%#q).Match(%#q) = (%v, %v) want (%v, %v)", tt.pattern, tt.s, result, err, tt.result, tt.err)
					return
				}
			}
		})
	}
}

func TestMultiMatcher(t *testing.T) {
	tests := map[string]Result{
		"aaa/bbb":                             Follow,
		"aaa/bbb/ccc":                         Follow,
		"aaa/bbb/ccc/ddd":                     Follow,
		"aaa/bbb/ccc/ddd/eee":                 NotMatched,
		"aaa/zzz":                             Follow,
		"aaa/zzz/ccc":                         Follow,
		"aaa/zzz/ccc/zzz":                     Follow,
		"aaa/zzz/ccc/zzz/eee":                 Matched,
		"aaa/zzz/zzz":                         Follow,
		"aaa/zzz/zzz/zzz/zzz":                 Follow,
		"aaa/zzz/zzz/zzz/zzz/zzz":             Follow,
		"aaa/zzz/zzz/zzz/ccc/zzz/zzz/ddd/eee": Matched,
		"zzz/aaa/zzz":                         NotMatched,
		"yyy/aaa/yyy":                         NotMatched,
	}

	includes := Multi(
		New("aaa/**/ccc/**/eee"),
		New("zzz/**"),
	)
	excludes := Multi(
		New("aaa/bbb/ccc/ddd/eee"),
		New("zzz/**"),
	)

	for path, tt := range tests {
		exclude, err := excludes.Match(path)
		if err != nil {
			t.Error(err)
		}

		result := exclude
		if result == Matched {
			result = NotMatched
		} else {
			result, err = includes.Match(path)
			if err != nil {
				t.Error(err)
			}
		}

		if result != tt {
			t.Errorf("path %q result was %v expected %v", path, result, tt)
		}
	}
}

func TestMatchFunc(t *testing.T) {
	tests := map[string]Result{
		"aaa/bbb":              Follow,
		"aaa/bbb/ccc/ddd/eee":  Matched,
		"aaaa/zzz/ccc/zzz/eee": Matched,
		"aa/zzz/ccc/zzz/eee":   NotMatched,
	}

	// a custom matcher that uses path.Match, but only succeeds if the path
	// segment is more than 2 characters.
	match := func(pattern, name string) (matched bool, err error) {
		matched, err = path.Match(pattern, name)
		if matched && len(name) > 2 {
			return true, err
		}
		return false, err
	}

	for path, tt := range tests {
		result, err := New("a*/**/ccc/**/eee", WithMatchFunc(match)).Match(path)
		if err != nil {
			t.Error(err)
		}

		if result != tt {
			t.Errorf("path %q result was %v expected %v", path, result, tt)
		}
	}
}

func TestMultiMatcherInvalid(t *testing.T) {
	_, err := Multi(
		New("abc"),
		New("[]a]"),
	).Match("abcdef")

	if err == nil {
		t.Errorf("include pattern was invalid, but not error was returned")
	}
}

func TestGlob(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}

	defer os.RemoveAll(dir)

	os.MkdirAll(filepath.Join(dir, "files", "dir1"), 0777)
	os.MkdirAll(filepath.Join(dir, "files", "dir2"), 0777)
	os.MkdirAll(filepath.Join(dir, "files", "dir3"), 0111)
	os.MkdirAll(filepath.Join(dir, "ignore", "dir4"), 0777)

	ioutil.WriteFile(filepath.Join(dir, "files", "dir1", "file1.txt"), []byte{}, 0600)
	ioutil.WriteFile(filepath.Join(dir, "files", "dir1", "file2.txt"), []byte{}, 0600)
	ioutil.WriteFile(filepath.Join(dir, "files", "dir2", "file3.ignore"), []byte{}, 0600)
	ioutil.WriteFile(filepath.Join(dir, "files", "dir3", "file4.txt"), []byte{}, 0600)
	ioutil.WriteFile(filepath.Join(dir, "ignore", "dir4", "file5.txt"), []byte{}, 0600)

	matches, err := Glob(context.Background(), dir, New("files/**/*.txt"))
	if err != nil {
		t.Error(err)
	}

	if len(matches) != 2 {
		t.Errorf("was expecting 2 files, got %v", len(matches))
	}
}

func TestGlobMultiMatcher(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}

	defer os.RemoveAll(dir)

	os.MkdirAll(filepath.Join(dir, "files", "dir1"), 0777)
	os.MkdirAll(filepath.Join(dir, "files", "dir2"), 0777)

	ioutil.WriteFile(filepath.Join(dir, "files", "dir1", "File1.txt"), []byte{}, 0600)
	ioutil.WriteFile(filepath.Join(dir, "files", "dir1", "File2.txt"), []byte{}, 0600)
	ioutil.WriteFile(filepath.Join(dir, "files", "dir2", "File3.txt"), []byte{}, 0600)

	matches, err := Glob(context.Background(), dir, Multi(
		New(strings.ToLower("files/DIR1/file1.txt")),
		New(strings.ToLower("files/DIR1/file2.txt")),
	), WithPathTransformer(strings.ToLower))
	if err != nil {
		t.Error(err)
	}

	if len(matches) != 2 {
		t.Errorf("was expecting 2 files, got %v", len(matches))
	}
}

var globDir = flag.String("globdir", runtime.GOROOT(), "The directory to use for glob benchmarks")
var globPattern = flag.String("globpattern", "pkg/**/*.go", "The pattern to use for glob benchmarks")

func BenchmarkGlob(b *testing.B) {
	b.ReportAllocs()

	m := New(*globPattern)
	for n := 0; n < b.N; n++ {
		_, err := Glob(context.Background(), *globDir, m)
		if err != nil {
			b.Error(err)
		}
	}
}

/*
func BenchmarkGlobWithDoublestarMatch(b *testing.B) {
	b.ReportAllocs()

	m := New(*globPattern, WithMatchFunc(doublestar.Match))
	for n := 0; n < b.N; n++ {
		_, err := Glob(context.Background(), *globDir, m)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDoublestarGlob(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		_, err := doublestar.Glob(filepath.Join(*globDir, *globPattern))
		if err != nil {
			b.Error(err)
		}
	}
}

func benchmarkWalkDoublestarMatch(b *testing.B) {
	matches := make(map[string]os.FileInfo)

	var m sync.Mutex

	err := walker.Walk(*globDir, func(pathname string, fi os.FileInfo) error {
		rel := strings.TrimPrefix(pathname, *globDir)
		rel = filepath.ToSlash(strings.TrimPrefix(rel, "/"))

		if rel == "" {
			return nil
		}

		match, err := doublestar.Match(*globPattern, rel)
		if err != nil {
			return err
		}

		if match {
			m.Lock()
			defer m.Unlock()
			matches[pathname] = fi
		}

		return nil
	})

	if err != nil {
		b.Error(err)
	}
}

func BenchmarkWalkDoublestarMatch(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		benchmarkWalkDoublestarMatch(b)
	}
}
*/
