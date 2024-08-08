package parser

import (
	"net/mail"
	"net/url"
	"os"
	"testing"

	"github.com/publicsuffix/list/tools/internal/domain"
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
		want               *List
		wantErrs           []error
	}{
		{
			name: "empty",
			psl:  byteLines(""),
			want: list(),
		},

		{
			name: "just_comments",
			psl: byteLines(
				"// This is an empty PSL file.",
				"",
				"// Here is a second comment.",
			),
			want: list(
				comment(0, "This is an empty PSL file."),
				comment(2, "Here is a second comment."),
			),
		},

		{
			name: "just_suffixes_in_block",
			psl: byteLines(
				"// ===BEGIN PRIVATE DOMAINS===",
				"",
				"example.com",
				"other.example.com",
				"*.example.org",
				"",
				"// ===END PRIVATE DOMAINS===",
			),
			want: list(
				section(0, 7, "PRIVATE DOMAINS",
					suffixes(2, 5, noInfo,
						suffix(2, "example.com"),
						suffix(3, "other.example.com"),
						wildcard(4, 5, "example.org"),
					),
				),
			),
		},

		{
			name: "empty_sections",
			psl: byteLines(
				"// ===BEGIN IMAGINARY DOMAINS===",
				"// ===END IMAGINARY DOMAINS===",
				"// ===BEGIN FAKE DOMAINS===",
				"// ===END FAKE DOMAINS===",
			),
			want: list(
				section(0, 2, "IMAGINARY DOMAINS"),
				section(2, 4, "FAKE DOMAINS"),
			),
		},

		{
			name: "missing_section_end",
			psl: byteLines(
				"// ===BEGIN ICANN DOMAINS===",
			),
			want: list(
				section(0, 1, "ICANN DOMAINS"),
			),
			wantErrs: []error{
				ErrUnclosedSection{section(0, 1, "ICANN DOMAINS")},
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
			want: list(
				section(0, 4, "ICANN DOMAINS"),
			),

			wantErrs: []error{
				ErrNestedSection{
					SourceRange: mkSrc(1, 3),
					Name:        "SECRET DOMAINS",
					Section:     section(0, 4, "ICANN DOMAINS"),
				},
			},
		},

		{
			name: "unknown_section_header",
			psl: byteLines(
				"// ===TRANSFORM DOMAINS===",
			),
			want: list(),
			wantErrs: []error{
				ErrUnknownSectionMarker{mkSrc(0, 1)},
			},
		},

		{
			name: "suffixes_with_section_marker_in_header",
			psl: byteLines(
				"// Just some suffixes",
				"// ===BEGIN ICANN DOMAINS===",
				"com",
				"org",
				"",
				"// ===END ICANN DOMAINS===",
			),
			want: list(
				comment(0, "Just some suffixes"),
				section(1, 6, "ICANN DOMAINS",
					suffixes(2, 4, noInfo,
						suffix(2, "com"),
						suffix(3, "org"),
					),
				),
			),
		},

		{
			name: "suffixes_with_section_markers_inline",
			psl: byteLines(
				"// ===BEGIN ICANN DOMAINS===",
				"// Just some suffixes",
				"com",
				"// ===BEGIN OTHER DOMAINS===",
				"org",
				"// ===END OTHER DOMAINS===",
				"net",
				"",
				"// ===END ICANN DOMAINS===",
			),
			want: list(
				section(0, 9, "ICANN DOMAINS",
					suffixes(1, 7,
						info("Just some suffixes", nil, nil, nil, true),
						comment(1, "Just some suffixes"),
						suffix(2, "com"),
						suffix(4, "org"),
						suffix(6, "net"),
					),
				),
			),
			wantErrs: []error{
				ErrSectionInSuffixBlock{mkSrc(3, 4)},
				ErrSectionInSuffixBlock{mkSrc(5, 6)},
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
			want: list(
				suffixes(0, 4,
					info(
						"Unstructured header.",
						nil,
						nil,
						[]string{"I'm just going on about random things."},
						true,
					),
					comment(0, "Unstructured header.", "I'm just going on about random things."),
					suffix(2, "example.com"),
					suffix(3, "example.org"),
				),
			),
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
			want: list(
				suffixes(0, 5,
					info(
						"DuckCorp Inc",
						urls("https://example.com"),
						emails("Not A Duck", "duck@example.com"),
						[]string{"Seriously, not a duck"},
						true),
					comment(0, "DuckCorp Inc: https://example.com", "Submitted by Not A Duck <duck@example.com>",
						"Seriously, not a duck"),
					suffix(3, "example.com"),
					suffix(4, "example.org"),
				),
			),
		},

		{
			name: "suffixes_with_entity_and_submitter",
			psl: byteLines(
				"// DuckCorp Inc: submitted by Not A Duck <duck@example.com>",
				"example.com",
			),
			want: list(
				suffixes(0, 2,
					info(
						"DuckCorp Inc",
						nil,
						emails("Not A Duck", "duck@example.com"),
						nil,
						true),
					comment(0, "DuckCorp Inc: submitted by Not A Duck <duck@example.com>"),
					suffix(1, "example.com"),
				),
			),
		},

		{
			name: "suffixes_with_all_separate_lines",
			psl: byteLines(
				"// DuckCorp Inc",
				"// https://example.com",
				"// Submitted by Not A Duck <duck@example.com>",
				"example.com",
			),
			want: list(
				suffixes(0, 4,
					info(
						"DuckCorp Inc",
						urls("https://example.com"),
						emails("Not A Duck", "duck@example.com"),
						nil,
						true),
					comment(0, "DuckCorp Inc", "https://example.com", `Submitted by Not A Duck <duck@example.com>`),
					suffix(3, "example.com"),
				),
			),
		},

		{
			// Regression test for a few blocks that start with "name
			// (url)" instead of the more common "name: url".
			name: "url_in_parens",
			psl: byteLines(
				"// Parens Appreciation Society (https://example.org)",
				"example.com",
			),
			want: list(
				suffixes(0, 2,
					info(
						"Parens Appreciation Society",
						urls("https://example.org"),
						nil,
						nil,
						true),
					comment(0, "Parens Appreciation Society (https://example.org)"),
					suffix(1, "example.com"),
				),
			),
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
			want: list(
				suffixes(0, 3,
					info(
						"cd",
						urls("https://en.wikipedia.org/wiki/.cd"),
						nil,
						[]string{"see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1"},
						true),
					comment(0, "cd : https://en.wikipedia.org/wiki/.cd",
						"see also: https://www.nic.cd/domain/insertDomain_2.jsp?act=1"),
					suffix(2, "cd"),
				),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, errs := Parse(test.psl)
			checkDiff(t, "parse result", got, test.want)
			checkDiff(t, "parse errors", errs, test.wantErrs)
		})
	}
}

// mkSrc returns a SourceRange with the given start and end.
func mkSrc(start, end int) SourceRange {
	return SourceRange{start, end}
}

// TestParseRealList checks that the real public suffix list can parse
// without errors.
func TestParseRealList(t *testing.T) {
	bs, err := os.ReadFile("../../../public_suffix_list.dat")
	if err != nil {
		t.Fatal(err)
	}

	_, errs := Parse(bs)

	for _, err := range errs {
		t.Errorf("Parse error: %v", err)
	}
}

func list(blocks ...Block) *List {
	return &List{
		Blocks: blocks,
	}
}

func comment(start int, lines ...string) *Comment {
	return &Comment{
		blockInfo: blockInfo{
			SourceRange: mkSrc(start, start+len(lines)),
		},
		Text: lines,
	}
}

func section(start, end int, name string, blocks ...Block) *Section {
	return &Section{
		blockInfo: blockInfo{
			SourceRange: mkSrc(start, end),
		},
		Name:   name,
		Blocks: blocks,
	}
}

func suffixes(start, end int, info MaintainerInfo, blocks ...Block) *Suffixes {
	return &Suffixes{
		blockInfo: blockInfo{
			SourceRange: mkSrc(start, end),
		},
		Info:   info,
		Blocks: blocks,
	}
}

func info(name string, urls []*url.URL, emails []*mail.Address, other []string, editable bool) MaintainerInfo {
	return MaintainerInfo{
		Name:            name,
		URLs:            urls,
		Maintainers:     emails,
		Other:           other,
		MachineEditable: editable,
	}
}

var noInfo = info("", nil, nil, nil, true)

func suffix(line int, domainStr string) *Suffix {
	domain, err := domain.Parse(domainStr)
	if err != nil {
		panic(err)
	}
	return &Suffix{
		blockInfo: blockInfo{
			SourceRange: mkSrc(line, line+1),
		},
		Domain: domain,
	}
}

func wildcard(start, end int, base string, exceptions ...string) *Wildcard {
	dom, err := domain.Parse(base)
	if err != nil {
		panic(err)
	}

	ret := &Wildcard{
		blockInfo: blockInfo{
			SourceRange: mkSrc(start, end),
		},
		Domain: dom,
	}
	for _, s := range exceptions {
		exc, err := domain.ParseLabel(s)
		if err != nil {
			panic(err)
		}
		ret.Exceptions = append(ret.Exceptions, exc)
	}
	return ret
}

// zeroSourceRange destructively zeroes the SourceRange of the given
// block and its children. We use a zero SourceRange to communicate
// "this block did not exist in the original input", when adding
// machine-generated blocks.
func zeroSourceRange(b Block) Block {
	switch v := b.(type) {
	case *List:
		v.SourceRange = SourceRange{}
	case *Section:
		v.SourceRange = SourceRange{}
	case *Suffixes:
		v.SourceRange = SourceRange{}
	case *Suffix:
		v.SourceRange = SourceRange{}
	case *Wildcard:
		v.SourceRange = SourceRange{}
	case *Comment:
		v.SourceRange = SourceRange{}
	default:
		panic("unknown ast node")
	}
	for _, child := range b.Children() {
		zeroSourceRange(child)
	}
	return b
}

// markUnchanged makes .Changed() return false for b. It does not
// touch parent or child blocks.
//
// It's generic so that it works in places that require a specific
// instance type, not just places that accept a Block interface.
func markUnchanged[T Block](b T) T {
	b.info().isUnchanged = true
	return b
}
