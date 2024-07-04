package parser

import (
	"testing"
)

func TestRequireSortedPrivateSection(t *testing.T) {
	aaa := suffixes(0, 1, "AAA Corp", "", "", suffix(0, "aaa.com"))
	bbb := suffixes(0, 1, "BBB Inc", "", "", suffix(0, "bbb.net"))
	ccc := suffixes(0, 1, "CCC Ltd", "", "", suffix(0, "ccc.org"))
	dddLeadingDot := suffixes(0, 1, ".DDD GmbH", "", "", suffix(0, "ddd.de"))
	aaaUmlaut := suffixes(0, 1, "AÄA", "", "", suffix(0, "aaa.de"))
	aaaUmlautShort := suffixes(0, 1, "AÄ", "", "", suffix(0, "aaa.ee"))
	aaaUmlautLong := suffixes(0, 1, "AÄAA", "", "", suffix(0, "aaa.sk"))
	a3b := suffixes(0, 1, "a3b", "", "", suffix(0, "a3b.com"))
	a24b := suffixes(0, 1, "a24b", "", "", suffix(0, "a24b.com"))

	tests := []struct {
		name string
		in   *Section
		want error
	}{
		{
			name: "easy_correct_order",
			in:   section(0, 0, "", aaa, bbb, ccc),
		},

		{
			name: "easy_wrong_order",
			// correct order: aaa, bbb, ccc
			in: section(0, 0, "", bbb, aaa, ccc),
			want: ErrSuffixBlocksInWrongPlace{
				EditScript: []MoveSuffixBlock{
					{
						Name:        bbb.Info.Name,
						InsertAfter: aaa.Info.Name,
					},
				},
			},
		},

		{
			name: "reversed",
			// correct order: aaa, bbb, ccc
			in: section(0, 0, "", ccc, bbb, aaa),
			want: ErrSuffixBlocksInWrongPlace{
				EditScript: []MoveSuffixBlock{
					{
						Name:        ccc.Info.Name,
						InsertAfter: aaa.Info.Name,
					},
					{
						Name:        bbb.Info.Name,
						InsertAfter: aaa.Info.Name,
					},
				},
			},
		},

		{
			name: "leading_punctuation",
			// correct order: dddLeadingDot, aaa, bbb, ccc
			in: section(0, 0, "", aaa, bbb, ccc, dddLeadingDot),
			want: ErrSuffixBlocksInWrongPlace{
				EditScript: []MoveSuffixBlock{
					{
						Name:        dddLeadingDot.Info.Name,
						InsertAfter: "",
					},
				},
			},
		},

		{
			name: "diacritics",
			// correct order: aaaUmlautShort, aaaUmlaut, aaa, aaaUmlautLong, bbb, ccc
			in: section(0, 0, "", aaa, bbb, ccc, aaaUmlaut, aaaUmlautShort, aaaUmlautLong),
			want: ErrSuffixBlocksInWrongPlace{
				EditScript: []MoveSuffixBlock{
					{
						Name:        aaaUmlaut.Info.Name,
						InsertAfter: "",
					},
					{
						Name:        aaaUmlautShort.Info.Name,
						InsertAfter: "",
					},
					{
						Name:        aaaUmlautLong.Info.Name,
						InsertAfter: aaa.Info.Name,
					},
				},
			},
		},

		{
			name: "numbers",
			// correct order: a24b, a3b, aaa, bbb
			in: section(0, 0, "", aaa, a3b, a24b, bbb),
			want: ErrSuffixBlocksInWrongPlace{
				EditScript: []MoveSuffixBlock{
					{
						Name:        aaa.Info.Name,
						InsertAfter: a24b.Info.Name,
					},
					{
						Name:        a3b.Info.Name,
						InsertAfter: a24b.Info.Name,
					},
				},
			},
		},

		{
			name: "amazon_superblock",
			in: section(0, 23, "",
				suffixes(2, 4, "AA Ltd", "", "", suffix(3, "aa.com")),

				comment(5, "Amazon : https://www.amazon.com", "several blocks follow"),
				// Note, incorrect sort, but ignored because it's in
				// the Amazon superblock.
				suffixes(8, 10, "eero", "", "", suffix(9, "eero.com")),
				suffixes(11, 13, "AWS", "", "", suffix(12, "aws.com")),
				comment(14, "concludes Amazon"),

				suffixes(16, 18, "Altavista", "", "", suffix(17, "altavista.com")),

				suffixes(19, 21, "BB Ltd", "", "", suffix(20, "bb.com")),
			),
			want: ErrSuffixBlocksInWrongPlace{
				EditScript: []MoveSuffixBlock{
					{
						Name:        `Amazon (all blocks until "concludes ..." comment)`,
						InsertAfter: "Altavista",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := validatePrivateSectionOrder(tc.in)
			checkDiff(t, "validation result", errs, tc.want)
		})
	}
}
