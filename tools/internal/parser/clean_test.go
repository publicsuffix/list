package parser

import (
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		name     string
		in, want *List
		wantErr  []error
	}{
		{
			name: "empty",
			in:   list(),
			want: list(),
		},

		{
			name: "toplevel_comments", // No change expected
			in: list(
				comment(1, "This is a comment", "one more"),
				comment(2, "separate comment"),
			),
			want: list(
				comment(1, "This is a comment", "one more"),
				comment(2, "separate comment"),
			),
		},

		{
			name: "sort_suffixes",
			in: list(
				suffixes(1, 1, noInfo,
					suffix(1, "com"),
					wildcard(2, 2, "foo.com"),
					suffix(3, "foo.com"),
					suffix(4, "qux.com"),
					suffix(5, "qux.foo.com"),
				),
			),
			want: list(
				suffixes(1, 1, noInfo,
					suffix(1, "com"),
					suffix(3, "foo.com"),
					wildcard(2, 2, "foo.com"),
					suffix(5, "qux.foo.com"),
					suffix(4, "qux.com"),
				),
			),
		},

		{
			name: "sort_suffixes_wildcard_after_suffix",
			in: list(
				suffixes(1, 1, noInfo,
					wildcard(1, 1, "foo.com"),
					suffix(2, "foo.com"),
					suffix(3, "bar.foo.com"),
				),
			),
			want: list(
				suffixes(1, 1, noInfo,
					suffix(2, "foo.com"),
					wildcard(1, 1, "foo.com"),
					suffix(3, "bar.foo.com"),
				),
			),
		},

		{
			name: "sort_suffixes_idna",
			in: list(
				// Taken from the xn--o3cw4h block of the PSL.
				suffixes(1, 1, noInfo,
					suffix(1, "ศึกษา.ไทย"),
					suffix(2, "ธุรกิจ.ไทย"),
					suffix(3, "รัฐบาล.ไทย"),
					suffix(4, "ทหาร.ไทย"),
					suffix(5, "เน็ต.ไทย"),
					suffix(6, "องค์กร.ไทย"),
					suffix(7, "com"),
					suffix(8, "africa"),
					suffix(9, "org"),
				),
			),
			want: list(
				// This is the correct sort order, as verified by a
				// Thai speaker/reader.
				//
				// Note that Latin script domains are sorted before
				// Thai script domains, because the PSL uses the
				// English language collation. In the Thai collation
				// the two scripts appear in the opposite order, but
				// the order within each script is the same.
				suffixes(1, 1, noInfo,
					suffix(8, "africa"),
					suffix(7, "com"),
					suffix(9, "org"),
					suffix(4, "ทหาร.ไทย"),
					suffix(2, "ธุรกิจ.ไทย"),
					suffix(5, "เน็ต.ไทย"),
					suffix(3, "รัฐบาล.ไทย"),
					suffix(1, "ศึกษา.ไทย"),
					suffix(6, "องค์กร.ไทย"),
				),
			),
		},

		{
			name: "sort_suffixes_dedup",
			in: list(
				suffixes(1, 1, noInfo,
					suffix(1, "com"),
					wildcard(2, 2, "foo.com"),
					suffix(3, "zot.com"),
					suffix(4, "qux.com"),
					suffix(5, "qux.foo.com"),
					suffix(6, "zot.com"),
				),
			),
			want: list(
				suffixes(1, 1, noInfo,
					suffix(1, "com"),
					wildcard(2, 2, "foo.com"),
					suffix(5, "qux.foo.com"),
					suffix(4, "qux.com"),
					suffix(3, "zot.com"),
				),
			),
		},

		{
			name: "sort_suffixes_with_nonblocking_comment",
			in: list(
				suffixes(1, 1, noInfo,
					// Both sides of the comment sorted wrong, but fixing
					// does not require going over the comment.
					suffix(1, "com"),
					suffix(2, "africa"),
					comment(3, "Random comment!"),
					suffix(4, "org"),
					suffix(5, "net"),
				),
			),
			want: list(
				suffixes(1, 1, noInfo,
					suffix(2, "africa"),
					suffix(1, "com"),
					comment(3, "Random comment!"),
					suffix(5, "net"),
					suffix(4, "org"),
				),
			),
		},

		{
			name: "sort_suffixes_with_blocking_comment",
			in: list(
				suffixes(1, 1, noInfo,
					// Both sides of the comment sorted wrong, but fixing
					// requires going over the comment.
					suffix(1, "org"),
					suffix(2, "net"),
					comment(3, "Random comment!"),
					suffix(4, "com"),
					suffix(5, "africa"),
				),
			),
			want: list(
				suffixes(1, 1, noInfo,
					suffix(2, "net"),
					suffix(1, "org"),
					comment(3, "Random comment!"),
					suffix(5, "africa"),
					suffix(4, "com"),
				),
			),
			wantErr: []error{
				ErrCommentPreventsSuffixSort{mkSrc(3, 4)},
			},
		},

		{
			name: "sort_suffixes_wildcard_exceptions",
			in: list(
				suffixes(1, 1, noInfo,
					// Also has a duplicate exception that needs cleaning up
					wildcard(1, 1, "foo.com", "mmm", "aaa", "zzz", "aaa"),
					suffix(2, "foo.com"),
				),
			),
			want: list(
				suffixes(1, 1, noInfo,
					suffix(2, "foo.com"),
					wildcard(1, 1, "foo.com", "aaa", "mmm", "zzz"),
				),
			),
		},

		{
			name: "sort_private_section_only",
			in: list(
				section(1, 1, "ICANN DOMAINS",
					suffixes(1, 1, info(".ZA", nil, nil, nil, true),
						comment(1, ".ZA"),
						suffix(2, "za"),
						suffix(3, "co.za"),
					),
					suffixes(2, 2, info(".BE", nil, nil, nil, true),
						comment(1, ".BE"),
						suffix(2, "com.be"),
						suffix(3, "be"),
					),
				),

				section(2, 2, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "zork.com"),
						suffix(3, "users.zork.com"),
					),
					suffixes(2, 2, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "advent"),
						suffix(3, "lamp"),
					),
				),
			),
			want: list(
				// Unchanged compared to above
				section(1, 1, "ICANN DOMAINS",
					suffixes(1, 1, info(".ZA", nil, nil, nil, true),
						comment(1, ".ZA"),
						suffix(2, "za"),
						suffix(3, "co.za"),
					),
					suffixes(2, 2, info(".BE", nil, nil, nil, true),
						comment(1, ".BE"),
						suffix(2, "com.be"),
						suffix(3, "be"),
					),
				),

				// Suffix blocks and suffixes are sorted.
				section(2, 2, "PRIVATE DOMAINS",
					suffixes(2, 2, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "advent"),
						suffix(3, "lamp"),
					),
					suffixes(1, 1, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "zork.com"),
						suffix(3, "users.zork.com"),
					),
				),
			),
		},

		{
			name: "sort_private_section_by_entity_name",
			in: list(
				// Blocks that are in the right order if you sort by
				// domain strings, but not if you sort by entity
				// names.
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "frobozz"),
						suffix(3, "hades"),
					),
					suffixes(2, 2, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "plugh"),
						suffix(3, "xyzzy"),
					),
				),
			),
			want: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(2, 2, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "plugh"),
						suffix(3, "xyzzy"),
					),
					suffixes(1, 1, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "frobozz"),
						suffix(3, "hades"),
					),
				),
			),
		},

		{
			name: "sort_private_section_unicode",
			in: list(
				// Block names in Ukranian
				section(1, 1, "PRIVATE DOMAINS",
					// "Hello"
					suffixes(1, 1, info("Привіт", nil, nil, nil, true),
						comment(1, "Привіт"),
						suffix(2, "bar"),
					),
					// "Sorry"
					suffixes(2, 2, info("Вибачте", nil, nil, nil, true),
						comment(1, "Вибачте"),
						suffix(2, "foo"),
					),
				),
			),
			want: list(
				section(1, 1, "PRIVATE DOMAINS",
					// "Sorry"
					suffixes(2, 2, info("Вибачте", nil, nil, nil, true),
						comment(1, "Вибачте"),
						suffix(2, "foo"),
					),
					// "Hello"
					suffixes(1, 1, info("Привіт", nil, nil, nil, true),
						comment(1, "Привіт"),
						suffix(2, "bar"),
					),
				),
			),
		},

		{
			name: "sort_private_section_nonblocking_comment",
			in: list(
				// Both sides of the comment sorted wrong, but fixing
				// does not require going over the comment.
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Curses", nil, nil, nil, true),
						comment(1, "Curses"),
						suffix(2, "bar"),
					),
					suffixes(2, 2, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "foo"),
					),
					comment(3, "Random comment"),
					suffixes(4, 4, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "zot"),
					),
					suffixes(5, 5, info("The Pawn", nil, nil, nil, true),
						comment(1, "The Pawn"),
						suffix(2, "qux"),
					),
				),
			),
			want: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(2, 2, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "foo"),
					),
					suffixes(1, 1, info("Curses", nil, nil, nil, true),
						comment(1, "Curses"),
						suffix(2, "bar"),
					),
					comment(3, "Random comment"),
					suffixes(5, 5, info("The Pawn", nil, nil, nil, true),
						comment(1, "The Pawn"),
						suffix(2, "qux"),
					),
					suffixes(4, 4, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "zot"),
					),
				),
			),
		},

		{
			name: "sort_private_section_blocking_comment",
			in: list(
				// Both sides of the comment sorted wrong, but fixing
				// requires going over the comment.
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "bar"),
					),
					suffixes(2, 2, info("Curses", nil, nil, nil, true),
						comment(1, "Curses"),
						suffix(2, "foo"),
					),
					comment(3, "Random comment"),
					suffixes(4, 4, info("The Pawn", nil, nil, nil, true),
						comment(1, "The Pawn"),
						suffix(2, "zot"),
					),
					suffixes(5, 5, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "qux"),
					),
				),
			),
			want: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(2, 2, info("Curses", nil, nil, nil, true),
						comment(1, "Curses"),
						suffix(2, "foo"),
					),
					suffixes(1, 1, info("Zork", nil, nil, nil, true),
						comment(1, "Zork"),
						suffix(2, "bar"),
					),
					comment(3, "Random comment"),
					suffixes(5, 5, info("Adventure", nil, nil, nil, true),
						comment(1, "Adventure"),
						suffix(2, "qux"),
					),
					suffixes(4, 4, info("The Pawn", nil, nil, nil, true),
						comment(1, "The Pawn"),
						suffix(2, "zot"),
					),
				),
			),
			wantErr: []error{
				ErrCommentPreventsSectionSort{mkSrc(3, 4)},
			},
		},

		{
			name: "sort_private_section_amazon_block",
			in: list(
				// Sorted correctly, despite the Amazon block in the
				// middle being sorted incorrectly relative to its
				// surroundings.
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("AAA", nil, nil, nil, true),
						comment(1, "AAA"),
						suffix(2, "bar"),
					),
					comment(2, "Amazon : https://www.amazon.com", "Several blocks follow"),
					suffixes(3, 3, info("AWS", nil, nil, nil, true),
						comment(1, "AWS"),
						suffix(2, "foo"),
					),
					suffixes(4, 4, info("eero", nil, nil, nil, true),
						comment(1, "eero"),
						suffix(2, "zot"),
					),
					comment(5, "concludes Amazon"),
					suffixes(6, 6, info("Aviating Ltd.", nil, nil, nil, true),
						comment(1, "Aviating Ltd."),
						suffix(2, "qux"),
					),
				),
			),
			want: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("AAA", nil, nil, nil, true),
						comment(1, "AAA"),
						suffix(2, "bar"),
					),
					comment(2, "Amazon : https://www.amazon.com", "Several blocks follow"),
					suffixes(3, 3, info("AWS", nil, nil, nil, true),
						comment(1, "AWS"),
						suffix(2, "foo"),
					),
					suffixes(4, 4, info("eero", nil, nil, nil, true),
						comment(1, "eero"),
						suffix(2, "zot"),
					),
					comment(5, "concludes Amazon"),
					suffixes(6, 6, info("Aviating Ltd.", nil, nil, nil, true),
						comment(1, "Aviating Ltd."),
						suffix(2, "qux"),
					),
				),
			),
		},

		{
			name: "rewrite_metadata_full_replace",
			in: list(
				suffixes(1, 1,
					info("New metadata",
						urls("https://example.com", "https://example.org"),
						emails("Ex Ample", "example@example.net"),
						[]string{"Complete example", "of rewrite"},
						true),
					comment(1, "mediocre previous metadata", "no useful info at all"),
					suffix(2, "foo.com"),
				),
			),
			want: list(
				suffixes(1, 1,
					info("New metadata",
						urls("https://example.com", "https://example.org"),
						emails("Ex Ample", "example@example.net"),
						[]string{"Complete example", "of rewrite"},
						true),
					zeroSourceRange(
						comment(1,
							"New metadata : https://example.com",
							"https://example.org",
							"Submitted by Ex Ample <example@example.net>",
							"Complete example",
							"of rewrite",
						),
					),
					suffix(2, "foo.com"),
				),
			),
		},

		{
			name: "rewrite_metadata_change_name",
			in: list(
				suffixes(1, 1,
					// Company name has changed
					info("Example & Sons est. 1903",
						urls("https://example.com", "https://example.org"),
						emails("Ex Ample", "example@example.net"),
						[]string{"Supplemental information here"},
						true),
					comment(1,
						"Example Ltd. : https://example.com",
						"https://example.org",
						"Submitted by Ex Ample <example@example.net>",
						"Supplemental information here",
					),
					suffix(2, "foo.com"),
				),
			),
			want: list(
				suffixes(1, 1,
					info("Example & Sons est. 1903",
						urls("https://example.com", "https://example.org"),
						emails("Ex Ample", "example@example.net"),
						[]string{"Supplemental information here"},
						true),
					zeroSourceRange(
						comment(1,
							"Example & Sons est. 1903 : https://example.com",
							"https://example.org",
							"Submitted by Ex Ample <example@example.net>",
							"Supplemental information here",
						),
					),
					suffix(2, "foo.com"),
				),
			),
		},

		{
			name: "rewrite_metadata_remove_first_url",
			in: list(
				suffixes(1, 1,
					info("Example Ltd.",
						// example.org, being the only remaining URL,
						// needs to move into the headline
						urls("https://example.org"),
						emails("Ex Ample", "example@example.net"),
						[]string{"Supplemental information here"},
						true),
					comment(1,
						"Example Ltd. : https://example.com",
						"https://example.org",
						"Submitted by Ex Ample <example@example.net>",
						"Supplemental information here",
					),
					suffix(2, "foo.com"),
				),
			),
			want: list(
				suffixes(1, 1,
					info("Example Ltd.",
						urls("https://example.org"),
						emails("Ex Ample", "example@example.net"),
						[]string{"Supplemental information here"},
						true),
					zeroSourceRange(
						comment(1,
							"Example Ltd. : https://example.org",
							"Submitted by Ex Ample <example@example.net>",
							"Supplemental information here",
						),
					),
					suffix(2, "foo.com"),
				),
			),
		},

		{
			name: "rewrite_metadata_add_maintainer",
			in: list(
				suffixes(1, 1,
					info("Example Ltd.",
						urls("https://example.com"),
						emails(
							"Ex Ample", "example@example.net",
							"Exempli Gratia", "example@example.va",
						),
						[]string{"Supplemental information here"},
						true),
					comment(1,
						"Example Ltd. : https://example.com",
						"Submitted by Ex Ample <example@example.net>",
						"Supplemental information here",
					),
					suffix(2, "foo.com"),
				),
			),
			want: list(
				suffixes(1, 1,
					info("Example Ltd.",
						urls("https://example.com"),
						emails(
							"Ex Ample", "example@example.net",
							"Exempli Gratia", "example@example.va",
						),
						[]string{"Supplemental information here"},
						true),
					zeroSourceRange(
						comment(1,
							"Example Ltd. : https://example.com",
							"Submitted by Ex Ample <example@example.net>",
							"Submitted by Exempli Gratia <example@example.va>",
							"Supplemental information here",
						),
					),
					suffix(2, "foo.com"),
				),
			),
		},

		{
			name: "rewrite_metadata_new_comment",
			in: list(
				suffixes(1, 1,
					info("Example Ltd.",
						urls("https://example.com"),
						emails("Ex Ample", "example@example.com"),
						nil,
						true),
					suffix(1, "foo.com"),
				),
			),
			want: list(
				suffixes(1, 1,
					info("Example Ltd.",
						urls("https://example.com"),
						emails("Ex Ample", "example@example.com"),
						nil,
						true),
					zeroSourceRange(
						comment(1,
							"Example Ltd. : https://example.com",
							"Submitted by Ex Ample <example@example.com>",
						),
					),
					suffix(1, "foo.com"),
				),
			),
		},

		{
			name: "rewrite_metadata_delete_comment",
			in: list(
				suffixes(1, 1, noInfo,
					comment(1,
						"Example Ltd. : https://example.com",
						"Submitted by Ex Ample <example@example.com>",
					),
					suffix(2, "foo.com"),
				),
			),
			want: list(
				suffixes(1, 1, noInfo,
					suffix(2, "foo.com"),
				),
			),
		},

		{
			name: "rewrite_metadata_non_editable",
			in: list(
				suffixes(1, 1,
					info(
						"Globotech",
						urls("https://example.org"),
						emails("Globo Technician", "globo@example.org"),
						[]string{"Amazing comment about Globotech's synergy"},
						false),
					comment(1,
						"Example Ltd. : https://example.com",
						"Submitted by Ex Ample <example@example.com>",
					),
					suffix(2, "foo.com"),
				),
			),
			want: list(
				// Info overwritten with raw info
				suffixes(1, 1,
					info(
						"Example Ltd.",
						urls("https://example.com"),
						emails("Ex Ample", "example@example.com"),
						nil,
						true),
					comment(1,
						"Example Ltd. : https://example.com",
						"Submitted by Ex Ample <example@example.com>",
					),
					suffix(2, "foo.com"),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			errs := got.Clean()
			checkDiff(t, "Clean result", got, tc.want)
			checkDiff(t, "Clean errors", errs, tc.wantErr)
		})
	}
}
