package domain_test

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/publicsuffix/list/tools/internal/domain"
	"golang.org/x/net/idna"
)

//go:generate go run update_idna_testdata.go

func TestParse(t *testing.T) {
	// This test is using the official Unicode IDNA test vectors, to
	// verify that domain.Parse is processing inputs exactly as
	// Unicode TR46 specifies. This is mostly a test of the behavior
	// of the underlying x/net/idna, but given the importance of
	// correctly validating public suffixes, we explicitly verify that
	// x/text/idna behaves correctly, and that our wrapper code
	// doesn't do anything surprising.

	numVectors := forEachIDNATestVector(t, func(input, want string, wantErr bool) {
		// PSL style deviates slightly from pure IDNA style, by
		// removing trailing dots if present. The removal is silent
		// because it doesn't affect the meaning of suffixes, but that
		// means the following tests have to allow for missing dots.
		//
		// Fortunately this adjustment does not break any of the IDNA
		// test vectors.
		wantNoTrailingDot := strings.TrimSuffix(want, ".")

		got, err := domain.Parse(input)
		gotErr := err != nil
		if gotErr != wantErr {
			t.Errorf("domain.Parse(%q) gotErr=%v, want %v", input, gotErr, wantErr)
			if err != nil {
				t.Logf("parse error was: %v", err)
			}
		}

		if err == nil && got.String() != wantNoTrailingDot {
			t.Errorf("domain.Parse(%q) = %q, want %q", input, got.String(), wantNoTrailingDot)
		}

		// Further tests only make sense on successful parses.
		if wantErr {
			return
		}

		// Domain parse succeeded, domain.ParseLabel of each label
		// must also succeed.
		//
		// We only do this for test vectors that don't return an
		// error, which means 'want' is in canonical form and '.' is
		// the only label separator character.
		var gotLabels []domain.Label
		for _, labelStr := range strings.Split(wantNoTrailingDot, ".") {
			label, err := domain.ParseLabel(labelStr)
			if err != nil {
				t.Errorf("domain.ParseLabel(%q) got err: %v", labelStr, err)
			} else {
				gotLabels = append(gotLabels, label)
			}
		}

		if wantLabels := got.Labels(); !slices.EqualFunc(gotLabels, wantLabels, domain.Label.Equal) {
			t.Error("domain.ParseLabel() of each label is not equivalent to ParseDomain().Labels()")
			t.Logf("domain.ParseLabel()    : %#v", gotLabels)
			t.Logf("domain.Parse().Labels(): %#v", wantLabels)
		}

		// ParseLabel must refuse to parse entire domains
		if got.NumLabels() > 1 {
			if gotLabel, err := domain.ParseLabel(input); err == nil {
				t.Errorf("domain.ParseLabel(%q) got %q, want parse error", input, gotLabel)
			}
		}

		// Domain and label comparisons are reflexive.
		if gotCmp := got.Compare(got); gotCmp != 0 {
			t.Errorf("Name.Compare(%q, %q) = %d, want 0", got, got, gotCmp)
		}
		for _, label := range gotLabels {
			if gotCmp := label.Compare(label); gotCmp != 0 {
				t.Errorf("Label.Compare(%q, %q) = %d, want 0", label, label, gotCmp)
			}
		}
	})
	t.Logf("checked %d test vectors", numVectors)

	// Sanity check to make sure the parser didn't just silently skip
	//  all test inputs. Manual inspection of the Unicode 15.0 test
	//  file shows 6235 tests. We allow a small amount of reduction
	//  because tests occasionally get removed (e.g. Unicode 15.1
	//  removes some vectors relating to deprecated special handling
	//  of "ß" in case mapping).
	const minVectors = 6200
	if numVectors < minVectors {
		t.Errorf("found %d test vectors, want at least %d", numVectors, minVectors)
	}
}

// forEachIDNATestVector parses testdata/idna_test_vectors.txt and
// calls fn in a subtest for each test vector. Return the number of
// test vectors found in the file.
func forEachIDNATestVector(t *testing.T, fn func(input, want string, wantErr bool)) (numVectorsFound int) {
	t.Helper()

	const testfile = "testdata/idna_test_vectors.txt"

	// Process the file in 2 passes. This is less efficient, it's
	// possible to stream the test file and do all this in one pass,
	// but the result is less readable.
	bs, err := os.ReadFile(testfile)
	if err != nil {
		t.Fatalf("reading IDNA test vectors: %v", err)
	}
	lines := strings.Split(string(bs), "\n")

	type testCase struct {
		line   int
		raw    string
		fields []string
	}
	var tests []testCase
	foundUnicodeVersion := false
	for i, ln := range lines {
		if ln == "" {
			continue
		}

		if unicodeVersion, ok := strings.CutPrefix(ln, "# Version: "); ok {
			if unicodeVersion != idna.UnicodeVersion {
				t.Fatalf("IDNA test file %q is for Unicode version %s, but x/net/idna uses version %s. Run 'go generate' to update the test file.", testfile, unicodeVersion, idna.UnicodeVersion)
			}
			foundUnicodeVersion = true
			continue
		}

		if strings.HasPrefix(ln, "#") {
			continue
		}

		fs := strings.Split(ln, "; ")
		if len(fs) != 7 {
			t.Fatalf("line %d: unrecognized test vector format: %s", i+1, ln)
		}
		tests = append(tests, testCase{i + 1, ln, fs})
	}
	if !foundUnicodeVersion {
		t.Fatalf("failed to determine Unicode version of test file, cannot proceed")
	}

	// Now we've collected all the test cases, prepare the inputs and
	// run the tests.
	for _, tc := range tests {
		input := tc.fields[0]
		want := tc.fields[1]
		wantErr := tc.fields[2] != ""

		// the input and want strings contain Unicode escape
		// sequences, so that the test can express precise invalid
		// inputs without risking accidental canonicalization by
		// editors and file readers. We have to carefully undo that
		// here, without making unwanted changes to the strings.
		input = unquoteVector(t, input)
		want = unquoteVector(t, want)

		// The test file format specifies that if the expected output
		// is the same as the input, they don't repeat it.
		if want == "" {
			want = input
		}

		t.Run(fmt.Sprintf("line_%d", tc.line), func(t *testing.T) {
			fn(input, want, wantErr)
			if t.Failed() {
				t.Logf("failing test vector: %s", tc.raw)
			}
		})
	}

	return len(tests)
}

// unquoteVector returns its input, with \uXXXX Unicode escape
// sequences converted to the corresponding UTF-8 bytes.
//
// In theory we could use strconv.Unquote, but that function handles
// more escape sequences that are not specified in the IDNA test
// format. Unquote may also mangle strings that are not valid UTF-8 in
// surprising ways, which could silently make tests check the wrong
// thing. To be safe, we do the unquoting ourselves, so that we are in
// full control of all mutations.
func unquoteVector(t *testing.T, s string) string {
	t.Helper()

	bs := []byte(s)
	var out []byte

	for {
		start, rest, found := bytes.Cut(bs, []byte(`\u`))
		out = append(out, start...)
		if !found {
			// No more escapes, we're done.
			break
		}

		// next 4 bytes are hex digits
		if len(rest) < 4 {
			t.Fatalf("malformed unicode escape sequence in %q", s)
		}
		hexStr := string(rest[:4])
		runeVal, err := strconv.ParseUint(hexStr, 16, 64)
		if err != nil {
			t.Fatalf("malformed unicode escape sequence in %q", s)
		}
		out = utf8.AppendRune(out, rune(runeVal))

		bs = rest[4:]
	}

	if !utf8.Valid(out) {
		t.Fatalf("string %q is invalid UTF-8 after unquote: %q", s, string(out))
	}
	return string(out)
}

func TestLabelCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"com", "com", 0},
		{"com", "org", -1},
		{"com", "aaa", +1},
		// Equivalent strings in NFC and NFD, ParseLabel should
		// canonicalize to equal.
		{"Québécois", "Que\u0301be\u0301cois", 0},
		// From the xn--o3cw4h block of the PSL.
		{"ทหาร", "ธุรกิจ", -1},
		{"ทหาร", "com", +1},
	}

	for _, tc := range tests {
		la, err := domain.ParseLabel(tc.a)
		if err != nil {
			t.Fatalf("ParseLabel(%q) failed: %v", tc.a, err)
		}
		lb, err := domain.ParseLabel(tc.b)
		if err != nil {
			t.Fatalf("ParseLabel(%q) failed: %v", tc.b, err)
		}

		gotCmp := domain.Label.Compare(la, lb)
		if gotCmp != tc.want {
			t.Errorf("Label.Compare(%q, %q) = %d, want %d", la, lb, gotCmp, tc.want)
		}
		wantEq := tc.want == 0
		if gotEq := domain.Label.Equal(la, lb); gotEq != wantEq {
			t.Errorf("Label.Equal(%q, %q) = %v, want %v", la, lb, gotEq, wantEq)
		}

		// Same again, but backwards.
		gotCmp = domain.Label.Compare(lb, la)
		if want := -tc.want; gotCmp != want {
			t.Errorf("Label.Compare(%q, %q) = %d, want %d", lb, la, gotCmp, want)
		}
		if gotEq := domain.Label.Equal(lb, la); gotEq != wantEq {
			t.Errorf("Label.Equal(%q, %q) = %v, want %v", lb, la, gotEq, wantEq)
		}
	}
}

func TestNameCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"foo.com", "foo.com.", 0},
		{"com", "org", -1},
		{"com", "aaa", +1},
		// Equivalent strings in NFC and NFD, ParseLabel should
		// canonicalize to equal.
		{"Québécois", "Que\u0301be\u0301cois", 0},
		// From the xn--o3cw4h block of the PSL.
		{"ทหาร", "ธุรกิจ", -1},
		{"ทหาร", "com", +1},
	}

	for _, tc := range tests {
		da, err := domain.Parse(tc.a)
		if err != nil {
			t.Fatalf("ParseLabel(%q) failed: %v", tc.a, err)
		}
		db, err := domain.Parse(tc.b)
		if err != nil {
			t.Fatalf("ParseLabel(%q) failed: %v", tc.b, err)
		}

		gotCmp := domain.Name.Compare(da, db)
		if gotCmp != tc.want {
			t.Errorf("Label.Compare(%q, %q) = %d, want %d", da, db, gotCmp, tc.want)
		}
		wantEq := tc.want == 0
		if gotEq := domain.Name.Equal(da, db); gotEq != wantEq {
			t.Errorf("Label.Equal(%q, %q) = %v, want %v", da, db, gotEq, wantEq)
		}

		// Same again, but backwards.
		gotCmp = domain.Name.Compare(db, da)
		if want := -tc.want; gotCmp != want {
			t.Errorf("Label.Compare(%q, %q) = %d, want %d", db, da, gotCmp, want)
		}
		if gotEq := domain.Name.Equal(db, da); gotEq != wantEq {
			t.Errorf("Label.Equal(%q, %q) = %v, want %v", db, da, gotEq, wantEq)
		}
	}
}
