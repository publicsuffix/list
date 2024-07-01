package parser

import (
	"bytes"
	"cmp"
	"net/mail"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	diff "github.com/google/go-cmp/cmp"
)

// TestParser runs a battery of synthetic parse and validation tests.
func TestParser(t *testing.T) {
	// These test cases have a fair amount of repetition in them,
	// since both errors and suffix blocks contain repeated nestings
	// of blocks and Source objects. While it's tempting to try and
	// reduce duplication through clever code, you are encouraged to
	// resist the urge.
	//
	// Each test case is quite verbose, but being laid out with
	// minimal indirection makes it easier to inspect and debug when a
	// failure happens.

	tests := []struct {
		name               string
		psl                []byte
		downgradeToWarning func(error) bool
		want               File
	}{
		{
			name: "empty",
			psl:  byteLines(""),
			want: File{},
		},

		{
			name: "just_comments",
			psl: byteLines(
				"// This is an empty PSL file.",
				"",
				"// Here is a second comment.",
			),
			want: File{
				Blocks: []Block{
					Comment{Source: mkSrc(0, "// This is an empty PSL file.")},
					Comment{Source: mkSrc(2, "// Here is a second comment.")},
				},
			},
		},

		{
			name: "just_suffixes",
			psl: byteLines(
				"example.com",
				"other.example.com",
				"*.example.org",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0, "example.com", "other.example.com", "*.example.org"),
						Entries: []Source{
							mkSrc(0, "example.com"),
							mkSrc(1, "other.example.com"),
							mkSrc(2, "*.example.org"),
						},
					},
				},
				Errors: []error{
					MissingEntityName{
						Suffixes: Suffixes{
							Source: mkSrc(0, "example.com", "other.example.com", "*.example.org"),
							Entries: []Source{
								mkSrc(0, "example.com"),
								mkSrc(1, "other.example.com"),
								mkSrc(2, "*.example.org"),
							},
						},
					},
				},
			},
		},

		{
			name: "empty_sections",
			psl: byteLines(
				"// ===BEGIN IMAGINARY DOMAINS===",
				"",
				"// ===END IMAGINARY DOMAINS===",
				"// ===BEGIN FAKE DOMAINS===",
				"// ===END FAKE DOMAINS===",
			),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: mkSrc(0, "// ===BEGIN IMAGINARY DOMAINS==="),
						Name:   "IMAGINARY DOMAINS",
					},
					EndSection{
						Source: mkSrc(2, "// ===END IMAGINARY DOMAINS==="),
						Name:   "IMAGINARY DOMAINS",
					},
					StartSection{
						Source: mkSrc(3, "// ===BEGIN FAKE DOMAINS==="),
						Name:   "FAKE DOMAINS",
					},
					EndSection{
						Source: mkSrc(4, "// ===END FAKE DOMAINS==="),
						Name:   "FAKE DOMAINS",
					},
				},
			},
		},

		{
			name: "missing_section_end",
			psl: byteLines(
				"// ===BEGIN ICANN DOMAINS===",
			),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: mkSrc(0, "// ===BEGIN ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
				},
				Errors: []error{
					UnclosedSectionError{
						Start: StartSection{
							Source: mkSrc(0, "// ===BEGIN ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
					},
				},
			},
		},

		{
			name: "nested_sections",
			psl: byteLines(
				"// ===BEGIN ICANN DOMAINS===",
				"// ===BEGIN SECRET DOMAINS===",
				"// ===END SECRET DOMAINS===",
				"// ===END ICANN DOMAINS===",
			),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: mkSrc(0, "// ===BEGIN ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
					StartSection{
						Source: mkSrc(1, "// ===BEGIN SECRET DOMAINS==="),
						Name:   "SECRET DOMAINS",
					},
					EndSection{
						Source: mkSrc(2, "// ===END SECRET DOMAINS==="),
						Name:   "SECRET DOMAINS",
					},
					EndSection{
						Source: mkSrc(3, "// ===END ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
				},
				Errors: []error{
					NestedSectionError{
						Outer: StartSection{
							Source: mkSrc(0, "// ===BEGIN ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
						Inner: StartSection{
							Source: mkSrc(1, "// ===BEGIN SECRET DOMAINS==="),
							Name:   "SECRET DOMAINS",
						},
					},
					UnstartedSectionError{
						EndSection{
							Source: mkSrc(3, "// ===END ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
					},
				},
			},
		},
		{
			name: "mismatched_sections",
			psl: byteLines(
				"// ===BEGIN ICANN DOMAINS===",
				"",
				"// ===END PRIVATE DOMAINS===",
			),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: mkSrc(0, "// ===BEGIN ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
					EndSection{
						Source: mkSrc(2, "// ===END PRIVATE DOMAINS==="),
						Name:   "PRIVATE DOMAINS",
					},
				},
				Errors: []error{
					MismatchedSectionError{
						Start: StartSection{
							Source: mkSrc(0, "// ===BEGIN ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
						End: EndSection{
							Source: mkSrc(2, "// ===END PRIVATE DOMAINS==="),
							Name:   "PRIVATE DOMAINS",
						},
					},
				},
			},
		},

		{
			name: "unknown_section_header",
			psl: byteLines(
				"// ===TRANSFORM DOMAINS===",
			),
			want: File{
				Blocks: []Block{
					Comment{
						Source: mkSrc(0, "// ===TRANSFORM DOMAINS==="),
					},
				},
				Errors: []error{
					UnknownSectionMarker{
						Line: mkSrc(0, "// ===TRANSFORM DOMAINS==="),
					},
				},
			},
		},

		{
			name: "suffixes_with_section_markers_in_header",
			psl: byteLines(
				"// Just some suffixes",
				"// ===BEGIN ICANN DOMAINS===",
				"com",
				"org",
				"",
				"// ===END ICANN DOMAINS===",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// Just some suffixes",
							"// ===BEGIN ICANN DOMAINS===",
							"com",
							"org",
						),
						Header: []Source{
							mkSrc(0, "// Just some suffixes"),
							mkSrc(1, "// ===BEGIN ICANN DOMAINS==="),
						},
						Entries: []Source{
							mkSrc(2, "com"),
							mkSrc(3, "org"),
						},
						Entity: "Just some suffixes",
					},
					EndSection{
						Source: mkSrc(5, "// ===END ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
				},
				Errors: []error{
					SectionInSuffixBlock{
						Line: mkSrc(1, "// ===BEGIN ICANN DOMAINS==="),
					},
					// Note: trying to gracefully parse the
					// StartSection would require splitting the suffix
					// block in two, which would need more code and
					// also result in additional spurious validation
					// errors. Instead this tests that section markers
					// within suffix blocks are ignored for section
					// validation.
					UnstartedSectionError{
						End: EndSection{
							Source: mkSrc(5, "// ===END ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
					},
				},
			},
		},

		{
			name: "suffixes_with_section_markers_inline",
			psl: byteLines(
				"// Just some suffixes",
				"com",
				"// ===BEGIN ICANN DOMAINS===",
				"org",
				"",
				"// ===END ICANN DOMAINS===",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// Just some suffixes",
							"com",
							"// ===BEGIN ICANN DOMAINS===",
							"org",
						),
						Header: []Source{
							mkSrc(0, "// Just some suffixes"),
						},
						Entries: []Source{
							mkSrc(1, "com"),
							mkSrc(3, "org"),
						},
						InlineComments: []Source{
							mkSrc(2, "// ===BEGIN ICANN DOMAINS==="),
						},
						Entity: "Just some suffixes",
					},
					EndSection{
						Source: mkSrc(5, "// ===END ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
				},
				Errors: []error{
					SectionInSuffixBlock{
						Line: mkSrc(2, "// ===BEGIN ICANN DOMAINS==="),
					},
					// Note: trying to gracefully parse the
					// StartSection would require splitting the suffix
					// block in two, which would need more code and
					// also result in additional spurious validation
					// errors. Instead this tests that section markers
					// within suffix blocks are ignored for section
					// validation.
					UnstartedSectionError{
						End: EndSection{
							Source: mkSrc(5, "// ===END ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
					},
				},
			},
		},

		{
			name: "suffixes_with_unstructured_header",
			psl: byteLines(
				"// Unstructured header.",
				"// I'm just going on about random things.",
				"example.com",
				"example.org",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// Unstructured header.",
							"// I'm just going on about random things.",
							"example.com",
							"example.org",
						),
						Header: []Source{
							mkSrc(0, "// Unstructured header."),
							mkSrc(1, "// I'm just going on about random things."),
						},
						Entries: []Source{
							mkSrc(2, "example.com"),
							mkSrc(3, "example.org"),
						},
						Entity: "Unstructured header.",
					},
				},
			},
		},

		{
			name: "suffixes_with_canonical_private_header",
			psl: byteLines(
				"// DuckCorp Inc: https://example.com",
				"// Submitted by Not A Duck <duck@example.com>",
				"// Seriously, not a duck",
				"example.com",
				"example.org",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// DuckCorp Inc: https://example.com",
							"// Submitted by Not A Duck <duck@example.com>",
							"// Seriously, not a duck",
							"example.com",
							"example.org",
						),
						Header: []Source{
							mkSrc(0, "// DuckCorp Inc: https://example.com"),
							mkSrc(1, "// Submitted by Not A Duck <duck@example.com>"),
							mkSrc(2, "// Seriously, not a duck"),
						},
						Entries: []Source{
							mkSrc(3, "example.com"),
							mkSrc(4, "example.org"),
						},
						Entity:    "DuckCorp Inc",
						URL:       mustURL("https://example.com"),
						Submitter: mustEmail("Not A Duck <duck@example.com>"),
					},
				},
			},
		},

		{
			name: "suffixes_with_entity_and_submitter",
			psl: byteLines(
				"// DuckCorp Inc: submitted by Not A Duck <duck@example.com>",
				"example.com",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// DuckCorp Inc: submitted by Not A Duck <duck@example.com>",
							"example.com",
						),
						Header: []Source{
							mkSrc(0, "// DuckCorp Inc: submitted by Not A Duck <duck@example.com>"),
						},
						Entries: []Source{
							mkSrc(1, "example.com"),
						},
						Entity:    "DuckCorp Inc",
						Submitter: mustEmail("Not A Duck <duck@example.com>"),
					},
				},
			},
		},

		{
			name: "suffixes_with_all_separate_lines",
			psl: byteLines(
				"// DuckCorp Inc",
				"// https://example.com",
				"// Submitted by Not A Duck <duck@example.com>",
				"example.com",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// DuckCorp Inc",
							"// https://example.com",
							"// Submitted by Not A Duck <duck@example.com>",
							"example.com",
						),
						Header: []Source{
							mkSrc(0, "// DuckCorp Inc"),
							mkSrc(1, "// https://example.com"),
							mkSrc(2, "// Submitted by Not A Duck <duck@example.com>"),
						},
						Entries: []Source{
							mkSrc(3, "example.com"),
						},
						Entity:    "DuckCorp Inc",
						URL:       mustURL("https://example.com"),
						Submitter: mustEmail("Not A Duck <duck@example.com>"),
					},
				},
			},
		},

		{
			name: "suffixes_standard_header_submitter_first",
			psl: byteLines(
				"// Submitted by Not A Duck <duck@example.com>",
				"// DuckCorp Inc: https://example.com",
				"example.com",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// Submitted by Not A Duck <duck@example.com>",
							"// DuckCorp Inc: https://example.com",
							"example.com",
						),
						Header: []Source{
							mkSrc(0, "// Submitted by Not A Duck <duck@example.com>"),
							mkSrc(1, "// DuckCorp Inc: https://example.com"),
						},
						Entries: []Source{
							mkSrc(2, "example.com"),
						},
						Entity:    "DuckCorp Inc",
						URL:       mustURL("https://example.com"),
						Submitter: mustEmail("Not A Duck <duck@example.com>"),
					},
				},
			},
		},

		{
			name: "suffixes_standard_header_leading_unstructured",
			psl: byteLines(
				"// This is an unstructured comment.",
				"// DuckCorp Inc: https://example.com",
				"// Submitted by Not A Duck <duck@example.com>",
				"example.com",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// This is an unstructured comment.",
							"// DuckCorp Inc: https://example.com",
							"// Submitted by Not A Duck <duck@example.com>",
							"example.com",
						),
						Header: []Source{
							mkSrc(0, "// This is an unstructured comment."),
							mkSrc(1, "// DuckCorp Inc: https://example.com"),
							mkSrc(2, "// Submitted by Not A Duck <duck@example.com>"),
						},
						Entries: []Source{
							mkSrc(3, "example.com"),
						},
						Entity:    "DuckCorp Inc",
						URL:       mustURL("https://example.com"),
						Submitter: mustEmail("Not A Duck <duck@example.com>"),
					},
				},
			},
		},

		{
			name: "legacy_error_downgrade",
			psl: byteLines(
				"// https://example.com",
				"example.com",
			),
			downgradeToWarning: func(e error) bool {
				return true
			},
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// https://example.com",
							"example.com",
						),
						Header: []Source{
							mkSrc(0, "// https://example.com"),
						},
						Entries: []Source{
							mkSrc(1, "example.com"),
						},
						URL: mustURL("https://example.com"),
					},
				},
				Warnings: []error{
					MissingEntityName{
						Suffixes: Suffixes{
							Source: mkSrc(0,
								"// https://example.com",
								"example.com",
							),
							Header: []Source{
								mkSrc(0, "// https://example.com"),
							},
							Entries: []Source{
								mkSrc(1, "example.com"),
							},
							URL: mustURL("https://example.com"),
						},
					},
				},
			},
		},

		{
			// Regression test for a few blocks that start with "name
			// (url)" instead of the more common "name: url".
			name: "url_in_parens",
			psl: byteLines(
				"// Parens Appreciation Society (https://example.org)",
				"example.com",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0, "// Parens Appreciation Society (https://example.org)", "example.com"),
						Header: []Source{
							mkSrc(0, "// Parens Appreciation Society (https://example.org)"),
						},
						Entries: []Source{
							mkSrc(1, "example.com"),
						},
						Entity: "Parens Appreciation Society",
						URL:    mustURL("https://example.org"),
					},
				},
			},
		},

		{
			// Regression test for a sneaky bug during development:
			// when an entity name is found when parsing Suffixes
			// headers, don't keep trying to find it in subsequent
			// lines, or you might overwrite the correct answer with
			// someething else that happens to have the right shape.
			name: "accept_first_valid_entity",
			psl: byteLines(
				"// cd : https://en.wikipedia.org/wiki/.cd",
				"// see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1",
				"cd",
			),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: mkSrc(0,
							"// cd : https://en.wikipedia.org/wiki/.cd",
							"// see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1",
							"cd",
						),
						Header: []Source{
							mkSrc(0, "// cd : https://en.wikipedia.org/wiki/.cd"),
							mkSrc(1, "// see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1"),
						},
						Entries: []Source{
							mkSrc(2, "cd"),
						},
						Entity: "cd",
						URL:    mustURL("https://en.wikipedia.org/wiki/.cd"),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exc := test.downgradeToWarning
			if exc == nil {
				// use real exceptions if the test doesn't provide something else
				exc = downgradeToWarning
			}
			got := parseWithExceptions(test.psl, exc, true).File
			checkDiff(t, "parse result", got, test.want)
		})
	}
}

// mustURL returns the given string as a URL, or panics if not a URL.
func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// mustEmail returns the given string as an RFC 5322 address, or
// panics if the parse fails.
func mustEmail(s string) *mail.Address {
	a, err := mail.ParseAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}

// mkSrc returns a Source with the given start, end, and dedented text.
func mkSrc(start int, lines ...string) Source {
	return Source{
		lineOffset: start,
		lines:      lines,
	}
}

// TestParseRealList checks that the real public suffix list can parse
// without errors.
func TestParseRealList(t *testing.T) {
	bs, err := os.ReadFile("../../../public_suffix_list.dat")
	if err != nil {
		t.Fatal(err)
	}

	f := Parse(bs)

	for _, err := range f.Errors {
		t.Errorf("Parse error: %v", err)
	}
}

// TestRoundtripRealList checks that concatenating the source text of
// all top-level blocks, with appropriate additional blank lines,
// exactly reproduces the source text that was parsed. Effectively,
// this is a "prove that the parser didn't discard any bytes" check.
func TestRoundtripRealList(t *testing.T) {
	bs, err := os.ReadFile("../../../public_suffix_list.dat")
	if err != nil {
		t.Fatal(err)
	}
	f := Parse(bs)

	if len(f.Errors) > 0 {
		t.Fatal("Parse errors, not attempting to roundtrip")
	}

	prevLine := 0
	var rebuilt bytes.Buffer
	for _, block := range f.Blocks {
		src := block.source()
		if src.lineOffset < prevLine {
			t.Fatalf("ordering error: previous block ended at %d but this block starts at %d:\n%s", prevLine, src.lineOffset, src.Text())
		}
		for prevLine < src.lineOffset {
			rebuilt.WriteByte('\n')
			prevLine++
		}
		rebuilt.WriteString(src.Text())
		rebuilt.WriteByte('\n')
		prevLine = src.lineOffset + len(src.lines)
	}

	got := strings.Split(strings.TrimSpace(rebuilt.String()), "\n")
	want := strings.Split(strings.TrimSpace(string(bs)), "\n")

	if diff := diff.Diff(want, got); diff != "" {
		t.Errorf("roundtrip failed (-want +got):\n%s", diff)
	}
}

// TestRoundtripRealListDetailed is like the prior round-tripping
// test, but Suffix blocks are written out using their
// Header/Entries/InlineComments fields, again as proof that no suffix
// block elements were lost during parsing.
func TestRoundtripRealListDetailed(t *testing.T) {
	bs, err := os.ReadFile("../../../public_suffix_list.dat")
	if err != nil {
		t.Fatal(err)
	}
	f := Parse(bs)

	if len(f.Errors) > 0 {
		t.Fatal("Parse errors, not attempting to roundtrip")
	}

	prevLine := 0
	var rebuilt bytes.Buffer
	for _, block := range f.Blocks {
		srcs := []Source{block.source()}
		if v, ok := block.(Suffixes); ok {
			srcs = []Source{}
			for _, h := range v.Header {
				srcs = append(srcs, h)
			}
			for _, e := range v.Entries {
				srcs = append(srcs, e)
			}
			for _, c := range v.InlineComments {
				srcs = append(srcs, c)
			}
			slices.SortFunc(srcs, func(a, b Source) int {
				return cmp.Compare(a.lineOffset, b.lineOffset)
			})
		}

		for _, src := range srcs {
			if src.lineOffset < prevLine {
				t.Fatalf("ordering error: previous block ended at %d but this block starts at %d:\n%s", prevLine, src.lineOffset, src.Text())
			}
			for prevLine < src.lineOffset {
				rebuilt.WriteByte('\n')
				prevLine++
			}
			rebuilt.WriteString(src.Text())
			rebuilt.WriteByte('\n')
			prevLine = src.lineOffset + len(src.lines)
		}
	}

	got := strings.Split(strings.TrimSpace(rebuilt.String()), "\n")
	want := strings.Split(strings.TrimSpace(string(bs)), "\n")

	if diff := diff.Diff(want, got); diff != "" {
		t.Errorf("roundtrip failed (-want +got):\n%s", diff)
	}
}

// TestExceptionsStillNecessary checks that all the exceptions in
// exeptions.go are still needed to parse the PSL without errors.
func TestExceptionsStillNecessary(t *testing.T) {
	bs, err := os.ReadFile("../../../public_suffix_list.dat")
	if err != nil {
		t.Fatal(err)
	}

	forEachOmitted(missingEmail, func(omitted string, trimmed []string) {
		old := missingEmail
		defer func() { missingEmail = old }()
		missingEmail = trimmed

		f := Parse(bs)
		if len(f.Errors) == 0 {
			t.Errorf("missingEmail exception no longer necessary:\n%s", omitted)
		}
	})
}

func forEachOmitted(exceptions []string, fn func(string, []string)) {
	for i := range exceptions {
		next := append([]string(nil), exceptions[:i]...)
		next = append(next, exceptions[i+1:]...)
		fn(exceptions[i], next)
	}
}
