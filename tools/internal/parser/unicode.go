package parser

import (
	"bytes"
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// How do you sort strings? The answer is surprisingly complex.
//
// "Collation" is the technical term for putting things in a specific
// order. For strings of human text, there is no universal agreement
// on what order is "correct".
//
// Different languages have different sorting conventions: in English
// Ä is an accented A and comes before B, but in Swedish Ä is the 28th
// letter of the alphabet and comes after Z.
//
// A single language also sorts differently sometimes: a phonebook
// written in the German language is in a slightly different order in
// Germany vs. Austria. Or even within a single country: in Germany, a
// list of names can be in "standard" order, or it can be in
// "phonebook" order, with different choices for ä, ö and ü.
//
// Finally, there are style choices available that are considered
// equally valid, depending on the application. A common example is
// "numeric sort", which order numbers inside strings according to
// mathematics: "3" > "24" in "standard" lexicographic order, but if a
// collation uses numeric sort, "3" < "24".
//
// Whitespace and punctuation are another example of a style choice:
// in some applications they participate in the ordering, and in
// others they are ignored and only "real" letters determine the
// order.
//
// Fortunately, the Unicode Consortium has simplified all this for us:
// there is a single universal Unicode Collation Algorithm
// (http://www.unicode.org/reports/tr10/) that handles all of this
// complexity. We just have to tell it which
// language/dialect/country/style we want to use, and now we can
// compare strings.
//
// For non-suffix text, the PSL uses the "basic" English
// collation. Specifically, we use the collation defined in the
// Unicode CLDR (Common Locale Data Repository,
// https://cldr.unicode.org/), described by the BCP 47 language tag
// "en": "global" English, with no country or dialect modifications,
// and "default" style choices for English: ordering is
// case-sensitive, whitespace-sensitive and punctuation-sensitive, and
// numbers are compared in lexicographic order, not numeric order.

// compareCommentText compares the strings of comment text a and b,
// using the PSL's chosen collation. It returns -1 if a < b, +1 if a >
// b, or 0 if a == b.
//
// This function MUST NOT be used to compare domain name or DNS label
// strings. For that, use domain.Name.Compare or domain.Label.Compare.
func compareCommentText(a string, b string) int {
	// golang.org/x/text/collate has a few bugs, and in particular the
	// "CompareString" method uses a special "incremental collation"
	// codepath that sometimes returns incorrect results (see
	// https://github.com/golang/go/issues/68166).
	//
	// To be safe, we instead use the "slower" (still pretty fast)
	// codepath: we explicitly convert the strings into the
	// corresponding "sort keys", and then bytes.Compare those. There
	// are more exhaustive tests for sort key computation, so there is
	// higher confidence that it works correctly.
	//
	// Unfortunately individual collators are also not safe for
	// concurrent use. Wrap them in a global mutex. We could also
	// construct a new collator for each use, but that ends up being
	// more expensive and less performant than sharing one collator
	// with a mutex.
	commentCollatorMu.Lock()
	defer commentCollatorMu.Unlock()
	var buf collate.Buffer
	ka := commentCollator.KeyFromString(&buf, a)
	kb := commentCollator.KeyFromString(&buf, b)
	return bytes.Compare(ka, kb)
}

// commentCollator compares strings in the PSL's chosen collation for
// non-suffix text. See the comment at the start of this file for more
// details.
var commentCollator = collate.New(language.MustParse("en"))
var commentCollatorMu sync.Mutex
