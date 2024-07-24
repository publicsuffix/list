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
