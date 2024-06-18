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

	wantLines := []source{
		mkSrc(0, "abc"),
		mkSrc(1, "def"),
		mkSrc(2, "ghi"),
		mkSrc(3, "jkl"),
	}
	checkDiff(t, "src.Lines()", src.Lines(), wantLines)

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
		src       source
		want      Source
		wantPanic bool
	}{
		{
			src:       mkSrc(0),
			wantPanic: true,
		},
		{
			src: mkSrc(0, "abc"),
			want: Source{
				StartLine: 0,
				EndLine:   1,
				Raw:       "abc",
			},
		},
		{
			src: mkSrc(0, "abc", "def"),
			want: Source{
				StartLine: 0,
				EndLine:   2,
				Raw:       "abc\ndef",
			},
		},
		{
			src: mkSrc(0, "abc", "def").line(0),
			want: Source{
				StartLine: 0,
				EndLine:   1,
				Raw:       "abc",
			},
		},
		{
			src: mkSrc(0, "abc", "def").line(1),
			want: Source{
				StartLine: 1,
				EndLine:   2,
				Raw:       "def",
			},
		},
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tc.wantPanic {
				defer func() {
					if err := recover(); err == nil {
						t.Errorf("wanted panic but got success")
					}
				}()
			}
			got := tc.src.Source()
			checkDiff(t, "Source()", got, tc.want)
			checkDiff(t, "Source().Raw vs. Text()", tc.src.Text(), got.Raw)
		})
	}
}

func TestForEachRun(t *testing.T) {
	t.Parallel()

	isComment := func(line source) bool {
		return strings.HasPrefix(line.Text(), "// ")
	}
	// some weird arbitrary classifier, to verify that ForEachRun is
	// using the classifier correctly
	groupCnt := 0
	groupsOf2And1 := func(line source) bool {
		groupCnt = (groupCnt + 1) % 3
		return groupCnt == 0
	}

	type Run struct {
		IsMatch bool
		Block   source
	}
	tests := []struct {
		name     string
		src      source
		classify func(source) bool
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
			tc.src.ForEachRun(tc.classify, func(block source, isMatch bool) {
				got = append(got, Run{isMatch, block})
			})
			checkDiff(t, "ForEachRun", got, tc.want)
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

	exact := func(s string) func(source) bool {
		return func(line source) bool {
			return line.Text() == s
		}
	}
	prefix := func(s string) func(source) bool {
		return func(line source) bool {
			return strings.HasPrefix(line.Text(), s)
		}
	}

	tests := []struct {
		name string
		src  source
		fn   func(source) bool
		want []source
	}{
		{
			name: "simple",
			src:  lines,
			fn:   exact("abc"),
			want: []source{
				mkSrc(0, "// comment"),
				mkSrc(2, "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "start",
			src:  lines,
			fn:   exact("// comment"),
			want: []source{
				mkSrc(1, "abc", "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "end",
			src:  lines,
			fn:   exact("ghi"),
			want: []source{
				mkSrc(0, "// comment", "abc", "", "// other", "def", "", "// end"),
			},
		},
		{
			name: "no_match",
			src:  lines,
			fn:   exact("xyz"),
			want: []source{
				mkSrc(0, "// comment", "abc", "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "prefix",
			src:  lines,
			fn:   prefix("ab"),
			want: []source{
				mkSrc(0, "// comment"),
				mkSrc(2, "", "// other", "def", "", "// end", "ghi"),
			},
		},
		{
			name: "prefix_comment",
			src:  lines,
			fn:   prefix("// "),
			want: []source{
				mkSrc(1, "abc", ""),
				mkSrc(4, "def", ""),
				mkSrc(7, "ghi"),
			},
		},

		{
			name: "empty",
			src:  mkSrc(0),
			fn:   exact("xyz"),
			want: []source{},
		},
		{
			name: "empty_split_blank",
			src:  mkSrc(0),
			fn:   exact(""),
			want: []source{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.src.Split(tc.fn)
			checkDiff(t, "Split", got, tc.want)
		})
	}
}

func TestCut(t *testing.T) {
	t.Parallel()

	exact := func(s string) func(source) bool {
		return func(line source) bool {
			return line.Text() == s
		}
	}
	prefix := func(s string) func(source) bool {
		return func(line source) bool {
			return strings.HasPrefix(line.Text(), s)
		}
	}

	tests := []struct {
		name         string
		src          source
		fn           func(source) bool
		before, rest source
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
			rest:   source{},
			found:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotBefore, gotRest, gotFound := tc.src.Cut(tc.fn)
			checkDiff(t, "Cut() before", gotBefore, tc.before)
			checkDiff(t, "Cut() after", gotRest, tc.rest)
			if gotFound != tc.found {
				t.Errorf("Cut() found=%v, want %v", gotFound, tc.found)
			}
		})
	}
}

func mkSrc(offset int, lines ...string) source {
	return source{
		lineOffset: offset,
		lines:      lines,
	}
}

func checkDiff(t *testing.T, whatIsBeingDiffed string, got, want any) {
	t.Helper()
	if diff := cmp.Diff(got, want, cmp.AllowUnexported(source{})); diff != "" {
		t.Errorf("%s is wrong (-got+want):\n%s", whatIsBeingDiffed, diff)
	}
}
