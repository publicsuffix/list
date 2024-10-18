package parser

import (
	"testing"

	"github.com/publicsuffix/list/tools/internal/domain"
)

func TestPublicSuffix(t *testing.T) {
	lst := list(
		section(1, 1, "PRIVATE DOMAINS",
			suffixes(1, 1, noInfo,
				suffix(1, "example.com"),
				wildcard(2, 3, "baz.net", "except", "other"),
				suffix(4, "com"),

				// Wildcards and exceptions nested inside each
				// other. This doesn't appear in the PSL in practice,
				// and is implicitly forbidden by the format spec, but
				// the parser/validator does not currently reject such
				// files, so we want PublicSuffix/RegisteredDomain to
				// be well-defined for such inputs.
				wildcard(5, 6, "nested.org", "except"),
				wildcard(7, 8, "in.except.nested.org", "other-except"),
			),
		),
	)

	tests := []struct {
		in        string
		pubSuffix string
		regDomain string
	}{
		{"www.example.com", "example.com", "www.example.com"},
		{"www.public.example.com", "example.com", "public.example.com"},
		{"example.com", "example.com", ""},

		{"www.other.com", "com", "other.com"},
		{"other.com", "com", "other.com"},
		{"com", "com", ""},

		{"qux.bar.baz.net", "bar.baz.net", "qux.bar.baz.net"},
		{"bar.baz.net", "bar.baz.net", ""},
		{"baz.net", "net", "baz.net"}, // Implicit * rule
		{"qux.except.baz.net", "baz.net", "except.baz.net"},
		{"except.baz.net", "baz.net", "except.baz.net"},
		{"other.other.baz.net", "baz.net", "other.baz.net"},

		// Tests for nested wildcards+exceptions. Does not appear in
		// the real PSL, and implicitly disallowed by the format spec,
		// but necessary to make PublicSuffix and RegisteredDomain's
		// outputs well defined for all inputs.
		{"qux.bar.foo.nested.org", "foo.nested.org", "bar.foo.nested.org"},
		{"bar.foo.nested.org", "foo.nested.org", "bar.foo.nested.org"},
		{"foo.nested.org", "foo.nested.org", ""},
		{"nested.org", "org", "nested.org"},
		{"bar.except.nested.org", "nested.org", "except.nested.org"},
		{"except.nested.org", "nested.org", "except.nested.org"},
		{"in.except.nested.org", "nested.org", "except.nested.org"},
		// Matches both nested wildcard and also outer exception,
		// outer exception wins.
		{"other.in.except.nested.org", "nested.org", "except.nested.org"},
		// Matches both outer and inner exceptions, inner exception
		// wins.
		{"qux.other-except.in.except.nested.org", "in.except.nested.org", "other-except.in.except.nested.org"},
	}

	for _, tc := range tests {
		in := mustParseDomain(tc.in)
		wantSuffix := mustParseDomain(tc.pubSuffix)

		gotSuffix := lst.PublicSuffix(in)
		if !gotSuffix.Equal(wantSuffix) {
			t.Errorf("PublicSuffix(%q) = %q, want %q", in, gotSuffix, wantSuffix)
		}

		gotReg, ok := lst.RegisteredDomain(in)
		if ok && tc.regDomain == "" {
			t.Errorf("RegisteredDomain(%q) = %q, want none", in, gotReg)
		} else if ok {
			wantReg := mustParseDomain(tc.regDomain)
			if !gotReg.Equal(wantReg) {
				t.Errorf("RegisteredDomain(%q) = %q, want %q", in, gotReg, wantReg)
			}
		}
	}
}

func mustParseDomain(s string) domain.Name {
	d, err := domain.Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}
