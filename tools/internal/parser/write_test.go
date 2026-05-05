package parser

import (
	"os"
	"testing"
)

func TestMarshalPSL(t *testing.T) {
	tests := []struct {
		name string
		in   *List
		want []byte
	}{
		{
			name: "empty",
			in:   list(),
			want: byteLines(""),
		},

		{
			name: "comments_and_empty_sections",
			in: list(
				comment(0, "This is a two", "line comment"),
				comment(0, "Another separate comment"),
				section(0, 0, "ICANN DOMAINS",
					comment(0, "Inside icann domains"),
				),
				comment(0, "Between sections"),
				section(0, 0, "PRIVATE DOMAINS",
					comment(0, "Private domains here"),
					comment(0, "More private domains"),
				),
			),
			want: byteLines(
				"// This is a two",
				"// line comment",
				"",
				"// Another separate comment",
				"",
				"// ===BEGIN ICANN DOMAINS===",
				"",
				"// Inside icann domains",
				"",
				"// ===END ICANN DOMAINS===",
				"",
				"// Between sections",
				"",
				"// ===BEGIN PRIVATE DOMAINS===",
				"",
				"// Private domains here",
				"",
				"// More private domains",
				"",
				"// ===END PRIVATE DOMAINS===",
				"",
			),
		},

		{
			name: "some_suffixes",
			in: list(
				comment(1, "Test list"),
				section(2, 2, "ICANN DOMAINS",
					suffixes(1, 1, noInfo,
						suffix(1, "aaa"),
						suffix(2, "bbb"),
						wildcard(3, 3, "ccc", "d", "e", "f"),
					),
					suffixes(2, 2, noInfo,
						suffix(1, "xxx"),
						suffix(2, "yyy"),
						suffix(3, "zzz"),
					),
				),
			),
			want: byteLines(
				"// Test list",
				"",
				"// ===BEGIN ICANN DOMAINS===",
				"",
				"aaa",
				"bbb",
				"*.ccc",
				"!d.ccc",
				"!e.ccc",
				"!f.ccc",
				"",
				"xxx",
				"yyy",
				"zzz",
				"",
				"// ===END ICANN DOMAINS===",
				"",
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.MarshalPSL()
			checkDiff(t, "MarhsalPSL output", got, tc.want)

			// Does the marshaled output parse?
			in2, errs := Parse(got)
			if len(errs) > 0 {
				t.Logf("failed to parse MarshalPSL output:")
				for _, err := range errs {
					t.Error(err)
				}
				t.FailNow()
			}

			// Parse result should be identical to the original,
			// modulo source ranges.
			zeroSourceRange(tc.in)
			zeroSourceRange(in2)
			checkDiff(t, "MarshalPSL then Parse", in2, tc.in)
			if t.Failed() {
				t.FailNow()
			}
		})
	}
}

func TestRoundtripRealPSL(t *testing.T) {
	bs, err := os.ReadFile("../../../public_suffix_list.dat")
	if err != nil {
		t.Fatal(err)
	}

	psl, errs := Parse(bs)
	if len(errs) > 0 {
		t.Logf("PSL parse failed, skipping round-trip test:")
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}

	suffixCnt1 := len(BlocksOfType[*Suffix](psl))
	wildCnt1 := len(BlocksOfType[*Wildcard](psl))
	if got, wantMin := suffixCnt1, 1000; got < wantMin {
		t.Fatalf("PSL doesn't have enough suffixes, got %d want at least %d", got, wantMin)
	}
	if got, wantMin := wildCnt1, 2; got < wantMin {
		t.Fatalf("PSL doesn't have enough wildcards, got %d want at least %d", got, wantMin)
	}

	bs2 := psl.MarshalPSL()
	psl2, errs := Parse(bs2)
	if len(errs) > 0 {
		t.Logf("PSL parse after MarshalPSL failed:")
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}

	suffixCnt2 := len(BlocksOfType[*Suffix](psl2))
	wildCnt2 := len(BlocksOfType[*Wildcard](psl2))
	if got, want := suffixCnt2, suffixCnt1; got != want {
		t.Errorf("MarshalPSL changed suffix count, got %d want %d", got, want)
	}
	if got, want := wildCnt2, wildCnt1; got != want {
		t.Errorf("MarshalPSL changed wildcard count, got %d want %d", got, want)
	}

	zeroSourceRange(psl)
	zeroSourceRange(psl2)
	checkDiff(t, "PSL roundtrip through MarshalPSL", psl2, psl)
}
