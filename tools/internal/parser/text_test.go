package parser

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		in       []byte
		want     []string
		wantErrs []error
	}{
		{
			name: "empty_input",
			in:   []byte{},
			want: []string{},
		},
		{
			name: "no_early_errors",
			in: byteLines(
				"// This is a small replica",
				"// of the PSL",
				"com",
				"net",
				"lol",
				"",
				"// End of file",
			),
			want: []string{
				"// This is a small replica",
				"// of the PSL",
				"com",
				"net",
				"lol",
				"",
				"// End of file",
			},
		},
		{
			name: "utf16be_input_with_bom",
			in:   utf16BigWithBOM("utf-16 text"),
			want: []string{"utf-16 text"},
		},
		{
			name: "utf16le_input_with_bom",
			in:   utf16LittleWithBOM("utf-16 text"),
			want: []string{"utf-16 text"},
		},
		{
			name:     "utf16be_input",
			in:       utf16Big("utf-16 text utf-16 text utf-16 text"),
			want:     []string{"utf-16 text utf-16 text utf-16 text"},
			wantErrs: []error{ErrInvalidEncoding{"UTF-16BE (guessed)"}},
		},
		{
			name:     "utf16le_input",
			in:       utf16Little("utf-16 text utf-16 text utf-16 text"),
			want:     []string{"utf-16 text utf-16 text utf-16 text"},
			wantErrs: []error{ErrInvalidEncoding{"UTF-16LE (guessed)"}},
		},
		{
			name: "utf8_with_bom",
			in:   utf8WithBOM("utf-8 text"),
			want: []string{"utf-8 text"},
		},
		{
			name: "utf8_with_garbage",
			// See https://en.wikipedia.org/wiki/UTF-8 for a
			// description of UTF-8 encoding, to help understand why
			// these inputs are invalid.
			//
			// The invalid patterns are immediately followed by more
			// valid characters, to verify exactly how normalization
			// mangles the bytes around an invalid sequence.
			in: byteLines(
				"normal UTF-8",
				// Illegal start bitpattern (5 leading bits set to 1)
				"bad1: \xF8abc",
				// First byte declares 3-byte character, but ends after 2 bytes
				"bad2: \xE0\xBFabc",
				// Continuation byte outside of a character
				"bad3: \xBFabc",
				// Ascii space (0x20) encoded non-minimally
				"bad4: \xC0\xA0abc",
				"this line is ok",
			),
			want: []string{
				"normal UTF-8",
				"bad1: \uFFFDabc",
				"bad2: \uFFFDabc",
				"bad3: \uFFFDabc",
				"bad4: \uFFFD\uFFFDabc",
				"this line is ok",
			},
			wantErrs: []error{
				ErrInvalidUnicode{mkSrc(1, 2)},
				ErrInvalidUnicode{mkSrc(2, 3)},
				ErrInvalidUnicode{mkSrc(3, 4)},
				ErrInvalidUnicode{mkSrc(4, 5)},
			},
		},
		{
			name: "dos_line_endings",
			in: byteLines(
				"normal file\r",
				"except the lines\r",
				"end like it's 1991"),
			want: []string{
				"normal file",
				"except the lines",
				"end like it's 1991",
			},
		},
		{
			name: "trailing_whitespace",
			in: byteLines(
				"a file  ",
				"with all kinds\t\t",
				" \r\t",
				// Strange "spaces": em space, ideographic space,
				// 4/18em medium mathematical space.
				"of trailing space\u2003\u3000\u205f",
				"and one good line",
			),
			want: []string{
				"a file",
				"with all kinds",
				"",
				"of trailing space",
				"and one good line",
			},
		},
		{
			name: "leading_whitespace",
			in: byteLines(
				"  a file",
				"\t\twith all kinds",
				" \r\t", // ensure this is reported as trailing, not leading
				// Strange "spaces": em space, ideographic space,
				// 4/18em medium mathematical space.
				"\u2003\u3000\u205fof leading space",
				"and one good line",
			),
			want: []string{
				"a file",
				"with all kinds",
				"",
				"of leading space",
				"and one good line",
			},
		},
		{
			name: "the_most_wrong_line",
			in:   byteLines("\xef\xbb\xbf  \t  // Hello\xc3\x28 very broken line\t  \r"),
			want: []string{"// Hello\uFFFD( very broken line"},
			wantErrs: []error{
				ErrInvalidUnicode{mkSrc(0, 1)},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines, errs := normalizeToUTF8Lines(tc.in)
			checkDiff(t, "newSource error set", errs, tc.wantErrs)
			checkDiff(t, "newSource result", lines, tc.want)
		})
	}
}

func byteLines(lines ...any) []byte {
	var ret [][]byte
	for _, ln := range lines {
		switch v := ln.(type) {
		case string:
			ret = append(ret, []byte(v))
		case []byte:
			ret = append(ret, v)
		default:
			panic(fmt.Sprintf("unhandled type %T for bytes()", ln))
		}
	}
	return bytes.Join(ret, []byte("\n"))
}

func encodeFromUTF8(s string, e encoding.Encoding) []byte {
	ret, err := e.NewEncoder().Bytes([]byte(s))
	if err != nil {
		// Only way this can happen is if the input isn't valid UTF-8,
		// and we don't do that in these tests.
		panic(err)
	}
	return ret
}

func utf16Big(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM))
}

func utf16BigWithBOM(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.BigEndian, unicode.UseBOM))
}

func utf16Little(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM))
}

func utf16LittleWithBOM(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM))
}

func utf8WithBOM(s string) []byte {
	return encodeFromUTF8(s, unicode.UTF8BOM)
}

func checkDiff(t *testing.T, whatIsBeingDiffed string, got, want any) {
	t.Helper()

	// cmp.Diff refuses to examine unexported fields by default. Tell
	// it that it's okay to look at unexported fields of blocks and
	// blockInfo, since we own those fields and want to include their
	// values in comparisons.
	exportInfo := cmp.Exporter(func(t reflect.Type) bool {
		if t.Kind() != reflect.Pointer {
			t = reflect.PointerTo(t)
		}

		if t.Elem() == reflect.TypeFor[blockInfo]() {
			return true
		}

		if t.Implements(reflect.TypeFor[Block]()) {
			return true
		}

		return false
	})
	if diff := cmp.Diff(got, want, exportInfo); diff != "" {
		t.Errorf("%s is wrong (-got+want):\n%s", whatIsBeingDiffed, diff)
	}
}
