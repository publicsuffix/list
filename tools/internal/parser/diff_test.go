package parser

import (
	"slices"
	"testing"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name      string
		before    *List
		now       *List
		expansive bool
		changed   *List
	}{
		{
			name: "new_suffix",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(1, "example.com"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(1, "example.com"),
						suffix(2, "example.net"),
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(2, "example.net"),
					),
				),
			),
		},

		{
			name: "new_wildcard",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(1, "example.com"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(1, "example.com"),
						wildcard(2, 2, "example.net"),
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						wildcard(2, 2, "example.net"),
					),
				),
			),
		},

		{
			name: "new_exception",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(1, "example.com"),
						wildcard(2, 2, "example.net", "exception1"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(1, "example.com"),
						wildcard(2, 2, "example.net", "exception1", "exception2"),
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, noInfo,
						wildcard(2, 2, "example.net", "exception1", "exception2"),
					),
				),
			),
		},

		{
			name: "new_suffix_block",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
					),
					suffixes(1, 1, info("Zork Ltd", nil, nil, nil, true),
						suffix(1, "zork.example.com"),
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Zork Ltd", nil, nil, nil, true),
						suffix(1, "zork.example.com"),
					),
				),
			),
		},

		{
			name: "new_section",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
					),
				),
			),
			now: list(
				section(2, 2, "ICANN DOMAINS",
					suffixes(1, 1, info("aaa", nil, nil, nil, true),
						suffix(1, "aaa"),
					),
				),
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
					),
				),
			),
			changed: list(
				section(2, 2, "ICANN DOMAINS",
					suffixes(1, 1, info("aaa", nil, nil, nil, true),
						suffix(1, "aaa"),
					),
				),
			),
		},

		{
			name: "change_suffixes",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						wildcard(3, 3, "example.net", "exception"),
						wildcard(4, 4, "example.edu", "faculty"),
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.gov"), // com -> gov
						suffix(2, "example.org"),
						wildcard(3, 3, "example.net", "exception2"), // exception -> exception2
						wildcard(4, 4, "w.example.edu", "faculty"),  // example.edu -> w.example.edu
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.gov"),
						wildcard(3, 3, "example.net", "exception2"),
						wildcard(4, 4, "w.example.edu", "faculty"),
					),
				),
			),
		},

		{
			name: "change_block_info",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1,
						info(
							"Example LLC",
							urls("https://example.com"),
							nil,
							nil,
							true),
						comment(1, "Example LLC: https://example.com"),
						suffix(2, "example.com"),
						suffix(3, "example.org"),
						wildcard(4, 4, "example.net", "exception"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1,
						info(
							"Example LLC",
							urls("https://example.org"), // different URL
							nil,
							nil,
							true),
						comment(1, "Example LLC: https://example.org"),
						suffix(2, "example.com"),
						suffix(3, "example.org"),
						wildcard(4, 4, "example.net", "exception"),
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1,
						info(
							"Example LLC",
							urls("https://example.org"),
							nil,
							nil,
							true),
						comment(1, "Example LLC: https://example.org"),
						// Suffixes not changed
					),
				),
			),
		},

		{
			name: "duplicates_in_new",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						suffix(3, "example.com"), // dupe
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
						suffix(4, "example.fr"), // dupe
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(3, "example.com"),
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(2, "example.fr"),
						suffix(4, "example.fr"),
					),
				),
			),
		},

		{
			name: "duplicates_in_old",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						suffix(3, "example.com"), // dupe
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
			),
			now: list(
				// no changes compared to before
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						suffix(3, "example.com"),
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
			),
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						// all dupes marked for recheck, despite no changes
						suffix(1, "example.com"),
						suffix(3, "example.com"),
					),
				),
			),
		},

		{
			name: "deletions",
			before: list(
				section(1, 1, "SUFFIX DELETES",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						suffix(3, "example.net"),
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
				section(2, 2, "SUFFIX BLOCK DELETES",
					suffixes(1, 1, info("aaa", nil, nil, nil, true),
						suffix(1, "aaa"),
						suffix(2, "bbb"),
					),
					suffixes(1, 1, info("bbb", nil, nil, nil, true),
						suffix(1, "ccc"),
					),
				),
				section(3, 3, "SECTION TO DELETE",
					suffixes(1, 1, info("delete me", nil, nil, nil, true),
						suffix(1, "delete-me.zork"),
					),
				),
			),
			now: list(
				section(1, 1, "SUFFIX DELETES",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						// deleted example.org
						suffix(3, "example.net"),
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
				section(2, 2, "SUFFIX BLOCK DELETES",
					// deleted aaa
					suffixes(1, 1, info("bbb", nil, nil, nil, true),
						suffix(1, "ccc"),
					),
				),
				// deleted section to delete
			),
			changed: list(
				section(1, 1, "SUFFIX DELETES",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true)),
				),
				section(2, 2, "SUFFIX BLOCK DELETES"),
			),
		},

		{
			name: "expand",
			before: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						suffix(3, "example.net"),
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
			),
			now: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						suffix(3, "example.biz"), // changed
					),
					suffixes(2, 2, info("Unchanged GmbH", nil, nil, nil, true),
						suffix(1, "example.de"),
						suffix(2, "example.fr"),
						suffix(3, "example.nl"),
					),
				),
			),
			expansive: true,
			changed: list(
				section(1, 1, "PRIVATE DOMAINS",
					suffixes(1, 1, info("Example LLC", nil, nil, nil, true),
						// all suffixes marked, not just the changed one
						suffix(1, "example.com"),
						suffix(2, "example.org"),
						suffix(3, "example.biz"),
					),
				),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.now
			got.SetBaseVersion(tc.before, tc.expansive)
			deleteUnchanged(got)
			checkDiff(t, "MarkUnchanged output", got, tc.changed)
		})
	}
}

func deleteUnchanged(b Block) {
	if !b.Changed() {
		return
	}

	switch v := b.(type) {
	case *List:
		v.Blocks = slices.DeleteFunc(v.Blocks, func(child Block) bool {
			return !child.Changed()
		})
		if len(v.Blocks) == 0 {
			v.Blocks = nil
		}
	case *Section:
		v.Blocks = slices.DeleteFunc(v.Blocks, func(child Block) bool {
			return !child.Changed()
		})
		if len(v.Blocks) == 0 {
			v.Blocks = nil
		}
	case *Suffixes:
		v.Blocks = slices.DeleteFunc(v.Blocks, func(child Block) bool {
			return !child.Changed()
		})
		if len(v.Blocks) == 0 {
			v.Blocks = nil
		}
	case *Suffix, *Wildcard, *Comment:
	default:
		panic("unknown ast node")
	}

	for _, child := range b.Children() {
		deleteUnchanged(child)
	}
}
