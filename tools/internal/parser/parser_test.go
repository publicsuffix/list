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
		psl                string
		downgradeToWarning func(error) bool
		want               File
	}{
		{
			name: "empty",
			psl:  "",
			want: File{},
		},

		{
			name: "just_comments",
			psl: dedent(`
			  // This is an empty PSL file.

			  // Here is a second comment.
			`),
			want: File{
				Blocks: []Block{
					Comment{Source: src(1, 1, "// This is an empty PSL file.")},
					Comment{Source: src(3, 3, "// Here is a second comment.")},
				},
			},
		},

		{
			name: "just_suffixes",
			psl: dedent(`
              example.com
              other.example.com
              *.example.org
			`),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 3, "example.com\nother.example.com\n*.example.org"),
						Entries: []Source{
							src(1, 1, "example.com"),
							src(2, 2, "other.example.com"),
							src(3, 3, "*.example.org"),
						},
					},
				},
				Errors: []error{
					MissingEntityName{
						Suffixes: Suffixes{
							Source: src(1, 3, "example.com\nother.example.com\n*.example.org"),
							Entries: []Source{
								src(1, 1, "example.com"),
								src(2, 2, "other.example.com"),
								src(3, 3, "*.example.org"),
							},
						},
					},
				},
			},
		},

		{
			name: "empty_sections",
			psl: dedent(`
			  // ===BEGIN IMAGINARY DOMAINS===

              // ===END IMAGINARY DOMAINS===
              // ===BEGIN FAKE DOMAINS===
              // ===END FAKE DOMAINS===
			`),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: src(1, 1, "// ===BEGIN IMAGINARY DOMAINS==="),
						Name:   "IMAGINARY DOMAINS",
					},
					EndSection{
						Source: src(3, 3, "// ===END IMAGINARY DOMAINS==="),
						Name:   "IMAGINARY DOMAINS",
					},
					StartSection{
						Source: src(4, 4, "// ===BEGIN FAKE DOMAINS==="),
						Name:   "FAKE DOMAINS",
					},
					EndSection{
						Source: src(5, 5, "// ===END FAKE DOMAINS==="),
						Name:   "FAKE DOMAINS",
					},
				},
			},
		},

		{
			name: "missing_section_end",
			psl: dedent(`
              // ===BEGIN ICANN DOMAINS===
            `),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: src(1, 1, "// ===BEGIN ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
				},
				Errors: []error{
					UnclosedSectionError{
						Start: StartSection{
							Source: src(1, 1, "// ===BEGIN ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
					},
				},
			},
		},

		{
			name: "nested_sections",
			psl: dedent(`
              // ===BEGIN ICANN DOMAINS===
              // ===BEGIN SECRET DOMAINS===
              // ===END SECRET DOMAINS===
              // ===END ICANN DOMAINS===
            `),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: src(1, 1, "// ===BEGIN ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
					StartSection{
						Source: src(2, 2, "// ===BEGIN SECRET DOMAINS==="),
						Name:   "SECRET DOMAINS",
					},
					EndSection{
						Source: src(3, 3, "// ===END SECRET DOMAINS==="),
						Name:   "SECRET DOMAINS",
					},
					EndSection{
						Source: src(4, 4, "// ===END ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
				},
				Errors: []error{
					NestedSectionError{
						Outer: StartSection{
							Source: src(1, 1, "// ===BEGIN ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
						Inner: StartSection{
							Source: src(2, 2, "// ===BEGIN SECRET DOMAINS==="),
							Name:   "SECRET DOMAINS",
						},
					},
					UnstartedSectionError{
						EndSection{
							Source: src(4, 4, "// ===END ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
					},
				},
			},
		},
		{
			name: "mismatched_sections",
			psl: dedent(`
              // ===BEGIN ICANN DOMAINS===

              // ===END PRIVATE DOMAINS===
            `),
			want: File{
				Blocks: []Block{
					StartSection{
						Source: src(1, 1, "// ===BEGIN ICANN DOMAINS==="),
						Name:   "ICANN DOMAINS",
					},
					EndSection{
						Source: src(3, 3, "// ===END PRIVATE DOMAINS==="),
						Name:   "PRIVATE DOMAINS",
					},
				},
				Errors: []error{
					MismatchedSectionError{
						Start: StartSection{
							Source: src(1, 1, "// ===BEGIN ICANN DOMAINS==="),
							Name:   "ICANN DOMAINS",
						},
						End: EndSection{
							Source: src(3, 3, "// ===END PRIVATE DOMAINS==="),
							Name:   "PRIVATE DOMAINS",
						},
					},
				},
			},
		},

		{
			name: "unknown_section_header",
			psl: dedent(`
              // ===TRANSFORM DOMAINS===
            `),
			want: File{
				Blocks: []Block{
					Comment{
						Source: src(1, 1, "// ===TRANSFORM DOMAINS==="),
					},
				},
				Errors: []error{
					UnknownSectionMarker{
						Line: src(1, 1, "// ===TRANSFORM DOMAINS==="),
					},
				},
			},
		},

		{
			name: "suffixes_with_unstructured_header",
			psl: dedent(`
              // Unstructured header.
              // I'm just going on about random things.
              example.com
              example.org
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 4, dedent(`
  						  // Unstructured header.
						  // I'm just going on about random things.
						  example.com
						  example.org`)),
						Header: []Source{
							src(1, 1, "// Unstructured header."),
							src(2, 2, "// I'm just going on about random things."),
						},
						Entries: []Source{
							src(3, 3, "example.com"),
							src(4, 4, "example.org"),
						},
						Entity: "Unstructured header.",
					},
				},
			},
		},

		{
			name: "suffixes_with_canonical_private_header",
			psl: dedent(`
              // DuckCorp Inc: https://example.com
              // Submitted by Not A Duck <duck@example.com>
              // Seriously, not a duck
              example.com
              example.org
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 5, dedent(`
  						  // DuckCorp Inc: https://example.com
                          // Submitted by Not A Duck <duck@example.com>
                          // Seriously, not a duck
						  example.com
						  example.org`)),
						Header: []Source{
							src(1, 1, "// DuckCorp Inc: https://example.com"),
							src(2, 2, "// Submitted by Not A Duck <duck@example.com>"),
							src(3, 3, "// Seriously, not a duck"),
						},
						Entries: []Source{
							src(4, 4, "example.com"),
							src(5, 5, "example.org"),
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
			psl: dedent(`
              // DuckCorp Inc: submitted by Not A Duck <duck@example.com>
              example.com
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 2, dedent(`
  						  // DuckCorp Inc: submitted by Not A Duck <duck@example.com>
						  example.com
                        `)),
						Header: []Source{
							src(1, 1, "// DuckCorp Inc: submitted by Not A Duck <duck@example.com>"),
						},
						Entries: []Source{
							src(2, 2, "example.com"),
						},
						Entity:    "DuckCorp Inc",
						Submitter: mustEmail("Not A Duck <duck@example.com>"),
					},
				},
			},
		},

		{
			name: "suffixes_with_all_separate_lines",
			psl: dedent(`
              // DuckCorp Inc
              // https://example.com
              // Submitted by Not A Duck <duck@example.com>
              example.com
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 4, dedent(`
  						  // DuckCorp Inc
                          // https://example.com
                          // Submitted by Not A Duck <duck@example.com>
						  example.com
                        `)),
						Header: []Source{
							src(1, 1, "// DuckCorp Inc"),
							src(2, 2, "// https://example.com"),
							src(3, 3, "// Submitted by Not A Duck <duck@example.com>"),
						},
						Entries: []Source{
							src(4, 4, "example.com"),
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
			psl: dedent(`
              // Submitted by Not A Duck <duck@example.com>
              // DuckCorp Inc: https://example.com
              example.com
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 3, dedent(`
                          // Submitted by Not A Duck <duck@example.com>
  						  // DuckCorp Inc: https://example.com
						  example.com
                        `)),
						Header: []Source{
							src(1, 1, "// Submitted by Not A Duck <duck@example.com>"),
							src(2, 2, "// DuckCorp Inc: https://example.com"),
						},
						Entries: []Source{
							src(3, 3, "example.com"),
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
			psl: dedent(`
              // This is an unstructured comment.
              // DuckCorp Inc: https://example.com
              // Submitted by Not A Duck <duck@example.com>
              example.com
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 4, dedent(`
                          // This is an unstructured comment.
  						  // DuckCorp Inc: https://example.com
                          // Submitted by Not A Duck <duck@example.com>
						  example.com
                        `)),
						Header: []Source{
							src(1, 1, "// This is an unstructured comment."),
							src(2, 2, "// DuckCorp Inc: https://example.com"),
							src(3, 3, "// Submitted by Not A Duck <duck@example.com>"),
						},
						Entries: []Source{
							src(4, 4, "example.com"),
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
			psl: dedent(`
              // https://example.com
              example.com
            `),
			downgradeToWarning: func(e error) bool {
				return true
			},
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 2, dedent(`
                          // https://example.com
						  example.com
                        `)),
						Header: []Source{
							src(1, 1, "// https://example.com"),
						},
						Entries: []Source{
							src(2, 2, "example.com"),
						},
						URL: mustURL("https://example.com"),
					},
				},
				Warnings: []error{
					MissingEntityName{
						Suffixes: Suffixes{
							Source: src(1, 2, dedent(`
                              // https://example.com
	    					  example.com
                            `)),
							Header: []Source{
								src(1, 1, "// https://example.com"),
							},
							Entries: []Source{
								src(2, 2, "example.com"),
							},
							URL: mustURL("https://example.com"),
						},
					},
				},
			},
		},

		{
			// Regression test for Future Versatile Group, who use a
			// unicode fullwidth colon in their header.
			name: "unicode_colon",
			psl: dedent(`
              // Future Versatile Group：https://example.org
              example.com
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 2, "// Future Versatile Group：https://example.org\nexample.com"),
						Header: []Source{
							src(1, 1, "// Future Versatile Group：https://example.org"),
						},
						Entries: []Source{
							src(2, 2, "example.com"),
						},
						Entity: "Future Versatile Group",
						URL:    mustURL("https://example.org"),
					},
				},
			},
		},

		{
			// Regression test for a few blocks that start with "name
			// (url)" instead of the more common "name: url".
			name: "url_in_parens",
			psl: dedent(`
              // Parens Appreciation Society (https://example.org)
              example.com
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 2, "// Parens Appreciation Society (https://example.org)\nexample.com"),
						Header: []Source{
							src(1, 1, "// Parens Appreciation Society (https://example.org)"),
						},
						Entries: []Source{
							src(2, 2, "example.com"),
						},
						Entity: "Parens Appreciation Society",
						URL:    mustURL("https://example.org"),
					},
				},
			},
		},

		{
			// Variation on the previous, some blocks in the "name
			// (url)" style don't have a scheme on their URL, so
			// require a bit more fudging to parse.
			name: "url_in_parens_no_scheme",
			psl: dedent(`
              // Parens Appreciation Society (hostyhosting.com)
              example.com

              // Parens Policy Panel (www.task.gda.pl/uslugi/dns)
              policy.example.org
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 2, "// Parens Appreciation Society (hostyhosting.com)\nexample.com"),
						Header: []Source{
							src(1, 1, "// Parens Appreciation Society (hostyhosting.com)"),
						},
						Entries: []Source{
							src(2, 2, "example.com"),
						},
						Entity: "Parens Appreciation Society",
						URL:    mustURL("https://hostyhosting.com"),
					},
					Suffixes{
						Source: src(4, 5, "// Parens Policy Panel (www.task.gda.pl/uslugi/dns)\npolicy.example.org"),
						Header: []Source{
							src(4, 4, "// Parens Policy Panel (www.task.gda.pl/uslugi/dns)"),
						},
						Entries: []Source{
							src(5, 5, "policy.example.org"),
						},
						Entity: "Parens Policy Panel",
						URL:    mustURL("https://www.task.gda.pl/uslugi/dns"),
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
			psl: dedent(`
              // cd : https://en.wikipedia.org/wiki/.cd
              // see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1
              cd
            `),
			want: File{
				Blocks: []Block{
					Suffixes{
						Source: src(1, 3, dedent(`
						  // cd : https://en.wikipedia.org/wiki/.cd
                          // see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1
                          cd`)),
						Header: []Source{
							src(1, 1, "// cd : https://en.wikipedia.org/wiki/.cd"),
							src(2, 2, "// see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1"),
						},
						Entries: []Source{
							src(3, 3, "cd"),
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
			got := parseWithExceptions(test.psl, exc)
			if diff := diff.Diff(&test.want, got); diff != "" {
				t.Errorf("unexpected parse result (-want +got):\n%s", diff)
			}
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

// src returns a Source with the given start, end, and dedented text.
func src(start, end int, text string) Source {
	return Source{
		StartLine: start,
		EndLine:   end,
		Raw:       dedent(text),
	}
}

// TestParseRealList checks that the real public suffix list can parse
// without errors.
func TestParseRealList(t *testing.T) {
	bs, err := os.ReadFile("../../../public_suffix_list.dat")
	if err != nil {
		t.Fatal(err)
	}

	f := Parse(string(bs))

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
	f := Parse(string(bs))

	if len(f.Errors) > 0 {
		t.Fatal("Parse errors, not attempting to roundtrip")
	}

	prevLine := 1
	var rebuilt bytes.Buffer
	for _, block := range f.Blocks {
		src := block.source()
		if src.StartLine < prevLine {
			t.Fatalf("ordering error: previous block ended at %d but this block starts at %d:\n%s", prevLine, src.StartLine, src.Raw)
		}
		for prevLine < src.StartLine {
			rebuilt.WriteByte('\n')
			prevLine++
		}
		rebuilt.WriteString(src.Raw)
		rebuilt.WriteByte('\n')
		prevLine = src.EndLine + 1
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
	f := Parse(string(bs))

	if len(f.Errors) > 0 {
		t.Fatal("Parse errors, not attempting to roundtrip")
	}

	prevLine := 1
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
				return cmp.Compare(a.StartLine, b.StartLine)
			})
		}

		for _, src := range srcs {
			if src.StartLine < prevLine {
				t.Fatalf("ordering error: previous block ended at %d but this block starts at %d:\n%s", prevLine, src.StartLine, src.Raw)
			}
			for prevLine < src.StartLine {
				rebuilt.WriteByte('\n')
				prevLine++
			}
			rebuilt.WriteString(src.Raw)
			rebuilt.WriteByte('\n')
			prevLine = src.EndLine + 1
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

		f := Parse(string(bs))
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
