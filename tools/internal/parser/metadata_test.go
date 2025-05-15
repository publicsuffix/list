package parser

import (
	"net/mail"
	"net/url"
	"testing"
)

func TestMetadata(t *testing.T) {
	tests := []struct {
		name string
		in   *Comment
		want MaintainerInfo
	}{
		{
			name: "empty",
			in:   nil,
			want: MaintainerInfo{
				MachineEditable: true,
			},
		},

		{
			name: "canonical",
			in: comment(0,
				"DuckCo : https://example.com",
				"Submitted by Duck <duck@example.com>",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("Duck", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "canonical_no_space_around_colon",
			in: comment(0,
				"DuckCo:https://example.com",
				"Submitted by Duck <duck@example.com>",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("Duck", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "canonical_url_in_parens",
			in: comment(0,
				"DuckCo (https://example.com)",
				"Submitted by Duck <duck@example.com>",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("Duck", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "canonical_by_registry",
			in: comment(0,
				"DuckCo : https://example.com",
				"Submitted by registry <duck@example.com>",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "name_and_email_first",
			in: comment(0,
				"DuckCo : Duck <duck@example.com>",
				"https://example.com",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("Duck", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "name_and_naked_email",
			in: comment(0,
				"DuckCo : duck@example.com",
				"https://example.com",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "one_per_line",
			in: comment(0,
				"DuckCo",
				"https://example.com",
				"Submitted by Duck <duck@example.com>",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("Duck", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "no_name",
			in: comment(0,
				"https://example.com",
				"Submitted by Duck <duck@example.com>",
				"Other notes here",
			),
			want: MaintainerInfo{
				Name:            "",
				URLs:            urls("https://example.com"),
				Maintainers:     emails("Duck", "duck@example.com"),
				Other:           []string{"Other notes here"},
				MachineEditable: true,
			},
		},

		{
			name: "http_url_and_bare_email",
			in: comment(0,
				"http://example.com",
				"duck@example.com",
			),
			want: MaintainerInfo{
				Name:            "",
				URLs:            urls("http://example.com"),
				Maintainers:     emails("", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "multiple_urls",
			in: comment(0,
				"DuckCo : https://example.com",
				"https://example.org/details",
				"Submitted by Duck <duck@example.com>",
			),
			want: MaintainerInfo{
				Name:            "DuckCo",
				URLs:            urls("https://example.com", "https://example.org/details"),
				Maintainers:     emails("Duck", "duck@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "multiple_emails",
			in: comment(0,
				"DuckCo : https://example.com",
				"Submitted by Duck <duck@example.com> and Goat <goat@example.com>",
				"llama@example.com",
			),
			want: MaintainerInfo{
				Name: "DuckCo",
				URLs: urls("https://example.com"),
				Maintainers: emails(
					"Duck", "duck@example.com",
					"Goat", "goat@example.com",
					"", "llama@example.com"),
				MachineEditable: true,
			},
		},

		{
			name: "multiple_everything_and_end_notes",
			in: comment(0,
				"DuckCo : https://example.com",
				"http://example.org",
				"https://example.net/more",
				"Submitted by Duck <duck@example.com> and Goat <goat@example.com>",
				"llama@example.com",
				`"Owl" <owl@example.net>`,
				"Duck is theoretically in charge, but Owl has influence",
				"Goat is not to be trusted, don't know about llama yet",
			),
			want: MaintainerInfo{
				Name: "DuckCo",
				URLs: urls("https://example.com", "http://example.org", "https://example.net/more"),
				Maintainers: emails(
					"Duck", "duck@example.com",
					"Goat", "goat@example.com",
					"", "llama@example.com",
					"Owl", "owl@example.net"),
				Other: []string{
					"Duck is theoretically in charge, but Owl has influence",
					"Goat is not to be trusted, don't know about llama yet",
				},
				MachineEditable: true,
			},
		},

		{
			name: "info_after_extra_notes",
			in: comment(0,
				"DuckCo",
				"Duck is in charge",
				"https://example.com",
				"Submitted by Duck <duck@example.com>",
			),
			want: MaintainerInfo{
				Name:        "DuckCo",
				URLs:        urls("https://example.com"),
				Maintainers: emails("Duck", "duck@example.com"),
				Other: []string{
					"Duck is in charge",
				},
				MachineEditable: false,
			},
		},

		{
			name: "obfuscated_email",
			in: comment(0,
				"lohmus",
				"someone at lohmus dot me",
			),
			want: MaintainerInfo{
				Name:            "lohmus",
				Maintainers:     emails("", "someone@lohmus.me"),
				MachineEditable: true,
			},
		},
	}

	for _, tc := range tests {
		got := extractMaintainerInfo(tc.in)
		checkDiff(t, "maintainer info", got, tc.want)
	}
}

func urls(us ...string) []*url.URL {
	var ret []*url.URL
	for _, s := range us {
		ret = append(ret, mustURL(s))
	}
	return ret
}

func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func emails(elts ...string) []*mail.Address {
	var ret []*mail.Address
	for i := 0; i < len(elts); i += 2 {
		ret = append(ret, email(elts[i], elts[i+1]))
	}
	return ret
}

func email(name, email string) *mail.Address {
	return &mail.Address{
		Name:    name,
		Address: email,
	}
}
