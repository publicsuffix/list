package parser

import (
	"testing"
)

func TestValidateEntityMetadata(t *testing.T) {
	in := list(
		section(1, 1, "PRIVATE DOMAINS",
			suffixes(1, 1, info("", nil, emails("Example", "example@example.com"), nil, true),
				comment(1, "Submitted by Example <example@example.com>"),
				suffix(2, "example.com"),
			),

			suffixes(2, 2, info("Example Ltd", nil, nil, nil, true),
				comment(1, "Example Ltd"),
				suffix(2, "example.org"),
			),

			suffixes(3, 3, noInfo,
				suffix(1, "example.net"),
			),

			suffixes(4, 4, info("Foo Ltd", nil, emails("Someone", "example@example.com"), nil, true),
				comment(1, "Submitted by Someone <example@example.com>"),
				suffix(2, "blah.example.com"),
			),
		),
	)
	want := []error{
		ErrMissingEntityName{
			Suffixes: suffixes(1, 1,
				info("", nil, emails("Example", "example@example.com"), nil, true),
				comment(1, "Submitted by Example <example@example.com>"),
				suffix(2, "example.com"),
			),
		},
		ErrMissingEntityEmail{
			Suffixes: suffixes(2, 2, info("Example Ltd", nil, nil, nil, true),
				comment(1, "Example Ltd"),
				suffix(2, "example.org"),
			),
		},
		ErrMissingEntityName{
			Suffixes: suffixes(3, 3, noInfo,
				suffix(1, "example.net"),
			),
		},
		ErrMissingEntityEmail{
			Suffixes: suffixes(3, 3, noInfo,
				suffix(1, "example.net"),
			),
		},
	}

	got := validateEntityMetadata(in)
	checkDiff(t, "validateEntityMetadata", got, want)

	// Make the change be a diff and check the reduced error set.
	prev := list(
		section(1, 1, "PRIVATE DOMAINS",
			suffixes(1, 1, info("", nil, emails("Example", "example@example.com"), nil, true),
				comment(1, "Submitted by Example <example@example.com>"),
				suffix(2, "example.com"),
			),

			suffixes(2, 2, info("Example Ltd", nil, nil, nil, true),
				comment(1, "Example Ltd"),
				suffix(2, "example.org"),
			),

			suffixes(3, 3, info("Foo Ltd", nil, emails("Someone", "example@example.com"), nil, true),
				comment(1, "Submitted by Someone <example@example.com>"),
				suffix(2, "blah.example.com"),
			),
		),
	)

	in.SetBaseVersion(prev, false)
	got = validateEntityMetadata(in)

	// Second suffix block no longer reports any errors. First one
	// still does, because its empty name is a dupe of the last block.
	want = []error{
		ErrMissingEntityName{
			Suffixes: suffixes(1, 1,
				info("", nil, emails("Example", "example@example.com"), nil, true),
				markUnchanged(comment(1, "Submitted by Example <example@example.com>")),
				markUnchanged(suffix(2, "example.com")),
			),
		},
		ErrMissingEntityName{
			Suffixes: suffixes(3, 3, noInfo,
				suffix(1, "example.net"),
			),
		},
		ErrMissingEntityEmail{
			Suffixes: suffixes(3, 3, noInfo,
				suffix(1, "example.net"),
			),
		},
	}

	checkDiff(t, "validateEntityMetadata (changed blocks only)", got, want)
}

func TestValidateExpectedSections(t *testing.T) {
	tests := []struct {
		name string
		in   *List
		want []error
	}{
		{
			name: "ok",
			in: list(
				section(1, 1, "ICANN DOMAINS"),
				section(2, 2, "PRIVATE DOMAINS"),
			),
			want: nil,
		},
		{
			name: "all_missing",
			in:   list(),
			want: []error{
				ErrMissingSection{"ICANN DOMAINS"},
				ErrMissingSection{"PRIVATE DOMAINS"},
			},
		},
		{
			name: "one_missing",
			in: list(
				section(1, 1, "ICANN DOMAINS"),
			),
			want: []error{
				ErrMissingSection{"PRIVATE DOMAINS"},
			},
		},
		{
			name: "unknown",
			in: list(
				section(1, 1, "ICANN DOMAINS"),
				section(2, 2, "PRIVATE DOMAINS"),
				section(3, 3, "NON EUCLIDEAN DOMAINS"),
			),
			want: []error{
				ErrUnknownSection{section(3, 3, "NON EUCLIDEAN DOMAINS")},
			},
		},
		{
			name: "duplicate_known",
			in: list(
				section(1, 1, "ICANN DOMAINS"),
				section(2, 2, "PRIVATE DOMAINS"),
				section(3, 3, "ICANN DOMAINS"),
			),
			want: []error{
				ErrDuplicateSection{
					section(3, 3, "ICANN DOMAINS"),
					section(1, 1, "ICANN DOMAINS"),
				},
			},
		},
		{
			name: "duplicate_unknown",
			in: list(
				section(1, 1, "RIDICULOUS DOMAINS"),
				section(2, 2, "ICANN DOMAINS"),
				section(3, 3, "PRIVATE DOMAINS"),
				section(4, 4, "RIDICULOUS DOMAINS"),
			),
			want: []error{
				ErrUnknownSection{section(1, 1, "RIDICULOUS DOMAINS")},
				ErrUnknownSection{section(4, 4, "RIDICULOUS DOMAINS")},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateExpectedSections(tc.in)
			checkDiff(t, "validateExpectedSections output", got, tc.want)
		})
	}
}

func TestValidateSuffixUniqueness(t *testing.T) {
	tests := []struct {
		name string
		in   *List
		want []error
	}{
		{
			name: "ok",
			in: list(
				section(1, 2, "PRIVATE DOMAINS",
					suffixes(2, 3, noInfo,
						suffix(3, "foo.com"),
						suffix(4, "bar.com"),
					),
				),
			),
			want: nil,
		},

		{
			name: "dupe_suffixes",
			in: list(
				section(1, 2, "PRIVATE DOMAINS",
					suffixes(2, 3, noInfo,
						suffix(3, "foo.com"),
						suffix(4, "bar.com"),
						suffix(5, "foo.com"),
					),
				),
			),
			want: []error{
				ErrDuplicateSuffix{"foo.com", suffix(5, "foo.com"), suffix(3, "foo.com")},
			},
		},

		{
			name: "dupe_wildcards",
			in: list(
				section(1, 2, "PRIVATE DOMAINS",
					suffixes(2, 3, noInfo,
						wildcard(3, 4, "foo.com"),
						suffix(4, "bar.com"),
						wildcard(5, 6, "foo.com"),
					),
				),
			),
			want: []error{
				ErrDuplicateSuffix{"*.foo.com", wildcard(5, 6, "foo.com"), wildcard(3, 4, "foo.com")},
			},
		},

		{
			name: "dupe_wildcard_exceptions",
			in: list(
				section(1, 2, "PRIVATE DOMAINS",
					suffixes(2, 3, noInfo,
						wildcard(3, 4, "foo.com", "a", "b", "c", "a"),
						suffix(4, "bar.com"),
						suffix(5, "b.foo.com"),
					),
				),
			),
			want: []error{
				ErrConflictingSuffixAndException{
					Suffix:   suffix(5, "b.foo.com"),
					Wildcard: wildcard(3, 4, "foo.com", "a", "b", "c", "a"),
				},
			},
		},

		{
			name: "dupe_spanning_blocks_and_sections",
			in: list(
				section(1, 2, "PRIVATE DOMAINS",
					suffixes(2, 3, noInfo,
						suffix(3, "foo.com"),
						suffix(4, "bar.com"),
					),
					suffixes(5, 6, noInfo,
						suffix(6, "foo.com"),
					),
				),
				section(7, 8, "ICANN DOMAINS",
					suffixes(8, 9, noInfo,
						suffix(9, "qux.com"),
						suffix(10, "foo.com"),
					),
				),
			),
			want: []error{
				ErrDuplicateSuffix{"foo.com", suffix(6, "foo.com"), suffix(3, "foo.com")},
				ErrDuplicateSuffix{"foo.com", suffix(10, "foo.com"), suffix(3, "foo.com")},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateSuffixUniqueness(tc.in)
			checkDiff(t, "validateSuffixUniqueness", got, tc.want)
		})
	}
}
