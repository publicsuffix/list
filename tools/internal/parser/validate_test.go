package parser

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestRequireSortedPrivateSection(t *testing.T) {
	// Shorthand for a simple suffix block with the right source data.
	suffixBlock := func(lineOffset int, name, suffix string) Suffixes {
		// For this test, every suffix block just has one suffix.
		src := mkSrc(lineOffset, fmt.Sprintf("// %s", name), suffix)
		return Suffixes{
			Source:  src,
			Header:  []Source{src.slice(0, 1)},
			Entries: []Source{src.slice(1, 2)},
			Entity:  name,
		}
	}
	// Shorthand for an input file containing a series of suffixes.
	suffixBlocks := func(suffixes ...Suffixes) []byte {
		var ret bytes.Buffer
		ret.WriteString("// ===BEGIN PRIVATE DOMAINS===\n\n")
		for _, block := range suffixes {
			for _, ln := range block.lineSources() {
				ret.WriteString(ln.Text())
				ret.WriteByte('\n')
			}
			ret.WriteByte('\n')
		}
		ret.WriteString("// ===END PRIVATE DOMAINS===\n")
		return ret.Bytes()
	}

	aaa := suffixBlock(0, "AAA Corp", "aaa.com")
	bbb := suffixBlock(0, "BBB Inc", "bbb.net")
	ccc := suffixBlock(0, "CCC Ltd", "ccc.org")
	dddLeadingDot := suffixBlock(0, ".DDD GmbH", "ddd.de")
	aaaUmlaut := suffixBlock(0, "AÄA", "aaa.de")
	aaaUmlautShort := suffixBlock(0, "AÄ", "aaa.ee")
	aaaUmlautLong := suffixBlock(0, "AÄAA", "aaa.sk")
	a3b := suffixBlock(0, "a3b", "a3b.com")
	a24b := suffixBlock(0, "a24b", "a24b.com")

	tests := []struct {
		name string
		in   []byte
		want []error
	}{
		{
			name: "easy_correct_order",
			in:   suffixBlocks(aaa, bbb, ccc),
		},
		{
			name: "easy_wrong_order",
			// correct order: aaa, bbb, ccc
			in: suffixBlocks(bbb, aaa, ccc),
			want: []error{
				SuffixBlocksInWrongPlace{
					EditScript: []MoveSuffixBlock{
						{
							Name:        bbb.Entity,
							InsertAfter: aaa.Entity,
						},
					},
				},
			},
		},
		{
			name: "reversed",
			// correct order: aaa, bbb, ccc
			in: suffixBlocks(ccc, bbb, aaa),
			want: []error{
				SuffixBlocksInWrongPlace{
					EditScript: []MoveSuffixBlock{
						{
							Name:        ccc.Entity,
							InsertAfter: aaa.Entity,
						},
						{
							Name:        bbb.Entity,
							InsertAfter: aaa.Entity,
						},
					},
				},
			},
		},
		{
			name: "leading_punctuation",
			// correct order: dddLeadingDot, aaa, bbb, ccc
			in: suffixBlocks(aaa, bbb, ccc, dddLeadingDot),
			want: []error{
				SuffixBlocksInWrongPlace{
					EditScript: []MoveSuffixBlock{
						{
							Name:        dddLeadingDot.Entity,
							InsertAfter: "",
						},
					},
				},
			},
		},
		{
			name: "diacritics",
			// correct order: aaaUmlautShort, aaaUmlaut, aaa, aaaUmlautLong, bbb, ccc
			in: suffixBlocks(aaa, bbb, ccc, aaaUmlaut, aaaUmlautShort, aaaUmlautLong),
			want: []error{
				SuffixBlocksInWrongPlace{
					EditScript: []MoveSuffixBlock{
						{
							Name:        aaaUmlaut.Entity,
							InsertAfter: "",
						},
						{
							Name:        aaaUmlautShort.Entity,
							InsertAfter: "",
						},
						{
							Name:        aaaUmlautLong.Entity,
							InsertAfter: aaa.Entity,
						},
					},
				},
			},
		},
		{
			name: "numbers",
			// correct order: a24b, a3b, aaa, bbb
			in: suffixBlocks(aaa, a3b, a24b, bbb),
			want: []error{
				SuffixBlocksInWrongPlace{
					EditScript: []MoveSuffixBlock{
						{
							Name:        aaa.Entity,
							InsertAfter: a24b.Entity,
						},
						{
							Name:        a3b.Entity,
							InsertAfter: a24b.Entity,
						},
					},
				},
			},
		},
		{
			name: "amazon_superblock",
			in: byteLines(
				"// ===BEGIN PRIVATE DOMAINS===",
				"",
				"// AA Ltd",
				"aa.com",
				"",
				"// Amazon : https://www.amazon.com",
				"// several blocks follow",
				"",
				// note: incorrect order, but ignored because in Amazon superblock
				"// eero",
				"eero.com",
				"",
				"// AWS",
				"aws.com",
				"",
				"// concludes Amazon",
				"",
				// note: out of order, not ignored
				"// Altavista",
				"altavista.com",
				"",
				"// BB Ltd",
				"bb.com",
				"",
				"// ===END PRIVATE DOMAINS===",
			),
			want: []error{
				SuffixBlocksInWrongPlace{
					EditScript: []MoveSuffixBlock{
						{
							Name:        `Amazon (all blocks until "concludes ..." comment)`,
							InsertAfter: "Altavista",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := parseWithExceptions(tc.in, downgradeToWarning, false)
			if len(p.File.Errors) > 0 {
				t.Fatalf("parse error before attempting validation: %v", errors.Join(p.File.Errors...))
			}
			p.requireSortedPrivateSection()

			checkDiff(t, "validation result", p.File.Errors, tc.want)
		})
	}
}
