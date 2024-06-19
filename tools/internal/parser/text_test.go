package parser

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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

func checkDiff(t *testing.T, whatIsBeingDiffed string, got, want any) {
	t.Helper()
	if diff := cmp.Diff(got, want, cmp.AllowUnexported(Source{})); diff != "" {
		t.Errorf("%s is wrong (-got+want):\n%s", whatIsBeingDiffed, diff)
	}
}
