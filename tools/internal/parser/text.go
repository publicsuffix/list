package parser

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	xunicode "golang.org/x/text/encoding/unicode"
)

// SourceRange describes a slice of lines from an unparsed source
// file. FirstLine and LastLine behave like normal slice offsets,
// i.e. they represent the half-open range [FirstLine:LastLine).
type SourceRange struct {
	FirstLine int
	LastLine  int
}

// NumLines returns the number of source lines described by
// SourceRange.
func (s SourceRange) NumLines() int {
	if s.FirstLine >= s.LastLine {
		return 0
	}
	return s.LastLine - s.FirstLine
}

// LocationString prints a human-readable description of the
// SourceRange.
func (s SourceRange) LocationString() string {
	switch {
	case s.LastLine <= s.FirstLine:
		return "<invalid SourceRange>"
	case s.LastLine == s.FirstLine+1:
		return fmt.Sprintf("line %d", s.FirstLine+1)
	default:
		return fmt.Sprintf("lines %d-%d", s.FirstLine+1, s.LastLine)
	}
}

// merge returns a SourceRange that contains both s and other. If s
// and other are not contiguous or overlapping, the returned
// SourceRange also spans unrelated lines, but always covers both s
// and other.
func (s SourceRange) merge(other SourceRange) SourceRange {
	return SourceRange{
		FirstLine: min(s.FirstLine, other.LastLine),
		LastLine:  max(s.LastLine, other.LastLine),
	}
}

// SrcRange returns the SourceRange. This looks a little strange, but
// it's to satisfy the Block interface. This allows other code to
// retrieve the SourceRange of any Block without having to typeswitch
// all the possible sub-types.
func (s SourceRange) SrcRange() SourceRange { return s }

const (
	bomUTF8    = "\xEF\xBB\xBF"
	bomUTF16BE = "\xFE\xFF"
	bomUTF16LE = "\xFF\xFE"
)

// The transformers that normalizeToUTF8Lines can use to process input
// into valid UTF-8, and that guessUTFVariant can return.
var (
	utf8Transform              = xunicode.UTF8BOM
	utf16LittleEndianTransform = xunicode.UTF16(xunicode.LittleEndian, xunicode.UseBOM)
	utf16BigEndianTransform    = xunicode.UTF16(xunicode.BigEndian, xunicode.UseBOM)
)

// normalizeToUTF8Lines slices bs into one string per line.
//
// All returned strings contain only valid UTF-8. Invalid byte
// sequences are replaced with the unicode replacement character
// (\uFFFD).
//
// The canonical PSL encoding is a file consisting entirely of valid
// UTF-8, with no leading BOM or unicode replacement characters. In an
// effort to report useful errors for common mangling caused by older
// Windows software, normalizeToUTF8Lines accepts input encoded as
// UTF-8, UTF-16LE or UTF-16BE, with or without a leading BOM.
//
// normalizeToUTF8Lines returns the normalized lines of bs, as well as
// errors that report deviations from the canonical encoding, if any.
func normalizeToUTF8Lines(bs []byte) ([]string, []error) {
	var errs []error

	enc := utf8Transform
	switch {
	case bytes.HasPrefix(bs, []byte(bomUTF8)):
		errs = append(errs, ErrUTF8BOM{})
	case bytes.HasPrefix(bs, []byte(bomUTF16BE)):
		enc = utf16BigEndianTransform
		errs = append(errs, ErrInvalidEncoding{"UTF-16BE"})
	case bytes.HasPrefix(bs, []byte(bomUTF16LE)):
		enc = utf16LittleEndianTransform
		errs = append(errs, ErrInvalidEncoding{"UTF-16LE"})
	default:
		enc = guessUTFVariant(bs)
		switch enc {
		case utf16BigEndianTransform:
			errs = append(errs, ErrInvalidEncoding{"UTF-16BE (guessed)"})
		case utf16LittleEndianTransform:
			errs = append(errs, ErrInvalidEncoding{"UTF-16LE (guessed)"})
		}
	}

	bs, err := enc.NewDecoder().Bytes(bs)
	if err != nil {
		// The decoder shouldn't error out, if it does we can't really
		// proceed, just return the errors we've found so far.
		errs = append(errs, err)
		return []string{}, errs
	}

	if len(bs) == 0 {
		return []string{}, errs
	}

	ret := strings.Split(string(bs), "\n")
	for i, line := range ret {
		// capture source info before we tidy up the line starts/ends,
		// so that input normalization errors show the problem being
		// described.
		//
		// However, we still provide post-sanitization UTF-8 bytes,
		// not the raw input. The raw input is unlikely to display
		// correctly in terminals and logs, and because the unicode
		// replacement character is a distinctive shape that stands
		// out, it should provide enough hints as to where any invalid
		// byte sequences are.
		src := SourceRange{i, i + 1}
		if strings.ContainsRune(line, utf8.RuneError) {
			errs = append(errs, ErrInvalidUTF8{src})
		}
		line, ok := strings.CutSuffix(line, "\r")
		if ok {
			ret[i] = line
			errs = append(errs, ErrDOSNewline{src})
		}
		if ln := strings.TrimRightFunc(line, unicode.IsSpace); ln != line {
			line = ln
			ret[i] = line
			errs = append(errs, ErrTrailingWhitespace{src})
		}
		if ln := strings.TrimLeftFunc(line, unicode.IsSpace); ln != line {
			line = ln
			ret[i] = line
			errs = append(errs, ErrLeadingWhitespace{src})
		}
	}

	return ret, errs
}

// guessUTFVariant guesses the encoding of bs.
//
// Returns the transformer to use on bs, one of utf8Transform,
// utf16LittleEndianTransform or utf16BigEndianTransform.
func guessUTFVariant(bs []byte) encoding.Encoding {
	// Only scan a few hundred bytes. Assume UTF-8 if we don't see
	// anything odd before that.
	const checkLimit = 200 // 100 UTF-16 characters
	if len(bs) > checkLimit {
		bs = bs[:checkLimit]
	}

	// This is a crude but effective trick to detect UTF-16: we assume
	// that the input contains at least some ascii, and that the
	// decoded input does not contain Unicode \u0000 codepoints
	// (legacy ascii null).
	//
	// If this is true, then valid UTF-8 text does not have any zero
	// bytes, because UTF-8 never produces a zero byte except when it
	// encodes the \u0000 codepoint.
	//
	// On the other hand, UTF-16 encodes all codepoints a pair of
	// bytes, and that means an ascii string in UTF-16 a zero byte
	// every 2 bytes. We can use the presence of zero bytes to
	// identify UTF-16, and the position of the zero (even or odd
	// offset) tells us what endianness to use.
	evenZeros, oddZeros := 0, 0
	for i, b := range bs {
		if b != 0 {
			continue
		}

		if i%2 == 0 {
			evenZeros++
		} else {
			oddZeros++
		}

		const (
			// Wait for a few zero bytes to accumulate, because if
			// this is just UTF-8 with a few \u0000 codepoints,
			// decoding as UTF-16 will be complete garbage. So, wait
			// until we see a suspicious number of zeros, and require
			// a strong bias towards even/odd before we guess
			// UTF-16. Otherwise, UTF-8 gives us the best chance of
			// producing coherent errors.
			decisionThreshold = 20
			utf16Threshold    = 15
		)
		if evenZeros+oddZeros < decisionThreshold {
			continue
		}
		if evenZeros > utf16Threshold {
			return utf16BigEndianTransform
		} else if oddZeros > utf16Threshold {
			return utf16LittleEndianTransform
		}
		// Lots of zeros, but no strong bias. No idea what's going on,
		// UTF-8 is a safe fallback.
		return utf8Transform
	}

	// Didn't find enough zeros, probably UTF-8.
	return utf8Transform
}
