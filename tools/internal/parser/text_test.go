package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		in       []byte
		want     []string
		wantErrs []error
	}{
		{
			name: "empty_input",
			in:   []byte{},
			want: []string{},
		},
		{
			name: "no_early_errors",
			in: byteLines(
				"// This is a small replica",
				"// of the PSL",
				"com",
				"net",
				"lol",
				"",
				"// End of file",
			),
			want: []string{
				"// This is a small replica",
				"// of the PSL",
				"com",
				"net",
				"lol",
				"",
				"// End of file",
			},
		},
		{
			name:     "utf16be_input_with_bom",
			in:       utf16BigWithBOM("utf-16 text"),
			want:     []string{"utf-16 text"},
			wantErrs: []error{InvalidEncodingError{"UTF-16BE"}},
		},
		{
			name:     "utf16le_input_with_bom",
			in:       utf16LittleWithBOM("utf-16 text"),
			want:     []string{"utf-16 text"},
			wantErrs: []error{InvalidEncodingError{"UTF-16LE"}},
		},
		{
			name:     "utf16be_input",
			in:       utf16Big("utf-16 text utf-16 text utf-16 text"),
			want:     []string{"utf-16 text utf-16 text utf-16 text"},
			wantErrs: []error{InvalidEncodingError{"UTF-16BE (guessed)"}},
		},
		{
			name:     "utf16le_input",
			in:       utf16Little("utf-16 text utf-16 text utf-16 text"),
			want:     []string{"utf-16 text utf-16 text utf-16 text"},
			wantErrs: []error{InvalidEncodingError{"UTF-16LE (guessed)"}},
		},
		{
			name:     "utf8_with_bom",
			in:       utf8WithBOM("utf-8 text"),
			want:     []string{"utf-8 text"},
			wantErrs: []error{UTF8BOMError{}},
		},
		{
			name: "utf8_with_garbage",
			// See https://en.wikipedia.org/wiki/UTF-8 for a
			// description of UTF-8 encoding, to help understand why
			// these inputs are invalid.
			//
			// The invalid patterns are immediately followed by more
			// valid characters, to verify exactly how normalization
			// mangles the bytes around an invalid sequence.
			in: byteLines(
				"normal UTF-8",
				// Illegal start bitpattern (5 leading bits set to 1)
				"bad1: \xF8abc",
				// First byte declares 3-byte character, but ends after 2 bytes
				"bad2: \xE0\xBFabc",
				// Continuation byte outside of a character
				"bad3: \xBFabc",
				// Ascii space (0x20) encoded non-minimally
				"bad4: \xC0\xA0abc",
				"this line is ok",
			),
			want: []string{
				"normal UTF-8",
				"bad1: \uFFFDabc",
				"bad2: \uFFFDabc",
				"bad3: \uFFFDabc",
				"bad4: \uFFFD\uFFFDabc",
				"this line is ok",
			},
			wantErrs: []error{
				InvalidUTF8Error{mkSrc(1, "bad1: \uFFFDabc")},
				InvalidUTF8Error{mkSrc(2, "bad2: \uFFFDabc")},
				InvalidUTF8Error{mkSrc(3, "bad3: \uFFFDabc")},
				InvalidUTF8Error{mkSrc(4, "bad4: \uFFFD\uFFFDabc")},
			},
		},
		{
			name: "dos_line_endings",
			in: byteLines(
				"normal file\r",
				"except the lines\r",
				"end like it's 1991"),
			want: []string{
				"normal file",
				"except the lines",
				"end like it's 1991",
			},
			wantErrs: []error{
				DOSNewlineError{
					Line: mkSrc(0, "normal file\r"),
				},
				DOSNewlineError{
					Line: mkSrc(1, "except the lines\r"),
				},
			},
		},
		{
			name: "trailing_whitespace",
			in: byteLines(
				"a file  ",
				"with all kinds\t\t",
				" \r\t",
				// Strange "spaces": em space, ideographic space,
				// 4/18em medium mathematical space.
				"of trailing space\u2003\u3000\u205f",
				"and one good line",
			),
			want: []string{
				"a file",
				"with all kinds",
				"",
				"of trailing space",
				"and one good line",
			},
			wantErrs: []error{
				TrailingWhitespaceError{
					Line: mkSrc(0, "a file  "),
				},
				TrailingWhitespaceError{
					Line: mkSrc(1, "with all kinds\t\t"),
				},
				TrailingWhitespaceError{
					Line: mkSrc(2, " \r\t"),
				},
				TrailingWhitespaceError{
					Line: mkSrc(3, "of trailing space\u2003\u3000\u205f"),
				},
			},
		},
		{
			name: "leading_whitespace",
			in: byteLines(
				"  a file",
				"\t\twith all kinds",
				" \r\t", // ensure this is reported as trailing, not leading
				// Strange "spaces": em space, ideographic space,
				// 4/18em medium mathematical space.
				"\u2003\u3000\u205fof leading space",
				"and one good line",
			),
			want: []string{
				"a file",
				"with all kinds",
				"",
				"of leading space",
				"and one good line",
			},
			wantErrs: []error{
				LeadingWhitespaceError{
					Line: mkSrc(0, "  a file"),
				},
				LeadingWhitespaceError{
					Line: mkSrc(1, "\t\twith all kinds"),
				},
				TrailingWhitespaceError{
					Line: mkSrc(2, " \r\t"),
				},
				LeadingWhitespaceError{
					Line: mkSrc(3, "\u2003\u3000\u205fof leading space"),
				},
			},
		},
		{
			name: "the_most_wrong_line",
			in:   byteLines("\xef\xbb\xbf  \t  // Hello\xc3\x28 very broken line\t  \r"),
			want: []string{"// Hello\uFFFD( very broken line"},
			wantErrs: []error{
				UTF8BOMError{},
				InvalidUTF8Error{
					Line: mkSrc(0, "  \t  // Hello\uFFFD( very broken line\t  \r"),
				},
				DOSNewlineError{
					Line: mkSrc(0, "  \t  // Hello\uFFFD( very broken line\t  \r"),
				},
				TrailingWhitespaceError{
					Line: mkSrc(0, "  \t  // Hello\uFFFD( very broken line\t  \r"),
				},
				LeadingWhitespaceError{
					Line: mkSrc(0, "  \t  // Hello\uFFFD( very broken line\t  \r"),
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src, errs := newSource(tc.in)
			checkDiff(t, "newSource error set", errs, tc.wantErrs)
			checkDiff(t, "newSource result", src.lines, tc.want)
		})
	}
}

func TestLineSlicing(t *testing.T) {
	t.Parallel()

	lines := []string{"abc", "def", "ghi", "jkl"}
	src := mkSrc(0, lines...)

	wantLines := []Source{
		mkSrc(0, "abc"),
		mkSrc(1, "def"),
		mkSrc(2, "ghi"),
		mkSrc(3, "jkl"),
	}
	checkDiff(t, "src.lineSources()", src.lineSources(), wantLines)

	// slice and line are internal helpers, but if they behave
	// incorrectly some higher level methods have very confusing
	// behavior, so test explicitly as well.
	for i, wantLine := range wantLines {
		checkDiff(t, fmt.Sprintf("src.line(%d)", i), src.line(i), wantLine)
	}

	for start := 0; start <= len(lines); start++ {
		for end := start + 1; end <= len(lines); end++ {
			t.Run(fmt.Sprintf("slice_%d_to_%d", start, end), func(t *testing.T) {
				want := mkSrc(start, lines[start:end]...)
				checkDiff(t, fmt.Sprintf("src.slice(%d, %d)", start, end), src.slice(start, end), want)
			})
		}
	}
}

func TestSourceText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src          Source
		wantText     string
		wantLocation string
	}{
		{
			src:          mkSrc(0),
			wantText:     "",
			wantLocation: "<invalid Source, 0-line range before line 1>",
		},
		{
			src:          mkSrc(0, "abc"),
			wantText:     "abc",
			wantLocation: "line 1",
		},
		{
			src:          mkSrc(0, "abc", "def"),
			wantText:     "abc\ndef",
			wantLocation: "lines 1-2",
		},
		{
			src:          mkSrc(0, "abc", "def").line(0),
			wantText:     "abc",
			wantLocation: "line 1",
		},
		{
			src:          mkSrc(0, "abc", "def").line(1),
			wantText:     "def",
			wantLocation: "line 2",
		},
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			checkDiff(t, "src.Text()", tc.src.Text(), tc.wantText)
			checkDiff(t, "mkSrc().LocationString()", tc.src.LocationString(), tc.wantLocation)
		})
	}
}

func TestForEachRun(t *testing.T) {
	t.Parallel()

	isComment := func(line Source) bool {
		return strings.HasPrefix(line.Text(), "// ")
	}
	// some weird arbitrary classifier, to verify that forEachRun is
	// using the classifier correctly
	groupCnt := 0
	groupsOf2And1 := func(line Source) bool {
		groupCnt = (groupCnt + 1) % 3
		return groupCnt == 0
	}

	type Run struct {
		IsMatch bool
		Block   Source
	}
	tests := []struct {
		name     string
		src      Source
		classify func(Source) bool
		want     []Run
	}{
		{
			name: "comments",
			src: mkSrc(0,
				"// foo",
				"// bar",
				"abc",
				"def",
				"// other",
				"ghi",
			),
			classify: isComment,
			want: []Run{
				{true, mkSrc(0, "// foo", "// bar")},
				{false, mkSrc(2, "abc", "def")},
				{true, mkSrc(4, "// other")},
				{false, mkSrc(5, "ghi")},
			},
		},
		{
			name: "only_comments",
			src: mkSrc(0,
				"// abc",
				"// def",
				"// ghi",
			),
			classify: isComment,
			want: []Run{
				{true, mkSrc(0, "// abc", "// def", "// ghi")},
			},
		},
		{
			name: "comment_at_end",
			src: mkSrc(0,
				"// abc",
				"def",
				"// ghi",
			),
			classify: isComment,
			want: []Run{
				{true, mkSrc(0, "// abc")},
				{false, mkSrc(1, "def")},
				{true, mkSrc(2, "// ghi")},
			},
		},
		{
			name: "no_comments",
			src: mkSrc(0,
				"abc",
				"def",
				"ghi",
			),
			classify: isComment,
			want: []Run{
				{false, mkSrc(0, "abc", "def", "ghi")},
			},
		},
		{
			name: "weird_classifier",
			src: mkSrc(0,
				"abc",
				"def",
				"ghi",
				"jkl",
				"mno",
				"pqr",
				"stu",
			),
			classify: groupsOf2And1,
			want: []Run{
				{false, mkSrc(0, "abc", "def")},
				{true, mkSrc(2, "ghi")},
				{false, mkSrc(3, "jkl", "mno")},
				{true, mkSrc(5, "pqr")},
				{false, mkSrc(6, "stu")}, // truncated final group
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []Run
			tc.src.forEachRun(tc.classify, func(block Source, isMatch bool) {
				got = append(got, Run{isMatch, block})
			})
			checkDiff(t, "forEachRun", got, tc.want)
		})
	}
}

func TestSplit(t *testing.T) {
	t.Parallel()

	lines := mkSrc(0,
		"// comment",
		"abc",
		"",
		"// other",
		"def",
		"",
		"// end",
		"ghi",
	)

	exact := func(s string) func(Source) bool {
		return func(line Source) bool {
			return line.Text() == s
		}
	}
	prefix := func(s string) func(Source) bool {
		return func(line Source) bool {
			return strings.HasPrefix(line.Text(), s)
		}
	}

	tests := []struct {
		name string
		src  Source
		fn   func(Source) bool
		want []Source
	}{
		{
			name: "simple",
			src:  lines,
			fn:   exact("abc"),
			want: []Source{
				mkSrc(0, "// comment"),
				mkSrc(2, "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "start",
			src:  lines,
			fn:   exact("// comment"),
			want: []Source{
				mkSrc(1, "abc", "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "end",
			src:  lines,
			fn:   exact("ghi"),
			want: []Source{
				mkSrc(0, "// comment", "abc", "", "// other", "def", "", "// end"),
			},
		},
		{
			name: "no_match",
			src:  lines,
			fn:   exact("xyz"),
			want: []Source{
				mkSrc(0, "// comment", "abc", "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "prefix",
			src:  lines,
			fn:   prefix("ab"),
			want: []Source{
				mkSrc(0, "// comment"),
				mkSrc(2, "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "prefix_comment",
			src:  lines,
			fn:   prefix("// "),
			want: []Source{
				mkSrc(1, "abc", ""),
				mkSrc(4, "def", ""),
				mkSrc(7, "ghi"),
			},
		},

		{
			name: "empty",
			src:  mkSrc(0),
			fn:   exact("xyz"),
			want: []Source{},
		},
		{
			name: "empty_split_blank",
			src:  mkSrc(0),
			fn:   exact(""),
			want: []Source{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.src.split(tc.fn)
			checkDiff(t, "split", got, tc.want)
		})
	}
}

func TestCut(t *testing.T) {
	t.Parallel()

	exact := func(s string) func(Source) bool {
		return func(line Source) bool {
			return line.Text() == s
		}
	}
	prefix := func(s string) func(Source) bool {
		return func(line Source) bool {
			return strings.HasPrefix(line.Text(), s)
		}
	}

	tests := []struct {
		name         string
		src          Source
		fn           func(Source) bool
		before, rest Source
		found        bool
	}{
		{
			name:   "simple",
			src:    mkSrc(0, "abc", "def", "ghi"),
			fn:     exact("def"),
			before: mkSrc(0, "abc"),
			rest:   mkSrc(1, "def", "ghi"),
			found:  true,
		},
		{
			name: "cut_on_first",
			src: mkSrc(0,
				"abc",
				"// def",
				"ghi",
				"// jkl",
				"mno",
			),
			fn:     prefix("// "),
			before: mkSrc(0, "abc"),
			rest:   mkSrc(1, "// def", "ghi", "// jkl", "mno"),
			found:  true,
		},
		{
			name:   "no_match",
			src:    mkSrc(0, "abc", "def", "ghi"),
			fn:     exact("xyz"),
			before: mkSrc(0, "abc", "def", "ghi"),
			rest:   Source{},
			found:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotBefore, gotRest, gotFound := tc.src.cut(tc.fn)
			checkDiff(t, "cut() before", gotBefore, tc.before)
			checkDiff(t, "cut() after", gotRest, tc.rest)
			if gotFound != tc.found {
				t.Errorf("cut() found=%v, want %v", gotFound, tc.found)
			}
		})
	}
}

func byteLines(lines ...any) []byte {
	var ret [][]byte
	for _, ln := range lines {
		switch v := ln.(type) {
		case string:
			ret = append(ret, []byte(v))
		case []byte:
			ret = append(ret, v)
		default:
			panic(fmt.Sprintf("unhandled type %T for bytes()", ln))
		}
	}
	return bytes.Join(ret, []byte("\n"))
}

func encodeFromUTF8(s string, e encoding.Encoding) []byte {
	ret, err := e.NewEncoder().Bytes([]byte(s))
	if err != nil {
		// Only way this can happen is if the input isn't valid UTF-8,
		// and we don't do that in these tests.
		panic(err)
	}
	return ret
}

func utf16Big(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM))
}

func utf16BigWithBOM(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.BigEndian, unicode.UseBOM))
}

func utf16Little(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM))
}

func utf16LittleWithBOM(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM))
}

func utf8WithBOM(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF8BOM)
}

func checkDiff(t *testing.T, whatIsBeingDiffed string, got, want any) {
	t.Helper()
	if diff := cmp.Diff(got, want, cmp.AllowUnexported(Source{})); diff != "" {
		t.Errorf("%s is wrong (-got+want):\n%s", whatIsBeingDiffed, diff)
	}
}
