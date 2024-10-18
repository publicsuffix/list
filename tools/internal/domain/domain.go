// Package domain provides parsing and processing of IDNA2008
// compliant domain names and DNS labels.
package domain

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"

	"golang.org/x/net/idna"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Name is a fully qualified domain name.
//
// A Name is always in valid, canonical Unicode form, according to the
// strictest ruleset for domain registration specified in IDNA2008 and
// Unicode Technical Standard #46.
type Name struct {
	// labels are the labels of the domain name.
	//
	// Within Name, labels are stored reversed compared to the string
	// form, e.g. "foo.com" becomes [com foo]. This is because the PSL
	// has to operate on suffixes a lot, but also do processing
	// label-wise not character-wise. This is much easier to do if the
	// slices of labels are stored in top-down order starting at the
	// TLD, instead of bottom-up from the leaf like in the
	// conventional string representation.
	//
	// Labels are always put back into the conventional leaf-first
	// order when returned to callers outside of Name.
	labels []Label
}

// Parse parses and validates a domain name string.
//
// s is validated and canonicalized into the Unicode form suitable for
// use in domain registrations, as defined by IDNA2008.
func Parse(s string) (Name, error) {
	// Note that some documentation around Unicode in domain names and
	// the PSL state that domain names must be normalized to NFKC. We
	// do not do this explicitly anywhere in this code, because
	// x/net/idna implements the TR46 algorithm
	// (https://www.unicode.org/reports/tr46/), and the mapping table
	// used in that algorithm implements NFKC in addition to other
	// validations and normalizations.
	//
	// So, if you found this comment because you were looking for NFKC
	// normalization logic, that's why you can't find it: this call to
	// ToUnicode is doing the work.
	canonical, err := domainValidator.ToUnicode(s)
	if err != nil {
		return Name{}, err
	}

	// Note we cannot split on "." first and then use ParseLabel here,
	// because ToUnicode canonicalizes several other dot-like
	// codepoints into ".". We have to canonicalize the whole string
	// first, then we can split it and construct the component Labels.
	labels := strings.Split(canonical, ".")
	if last := len(labels) - 1; last >= 0 && labels[last] == "" {
		// Canonical IDNA form allows one trailing dot, whereas PSL
		// convention is to omit the dot. This does not change the
		// meaning of the suffix in PSL terms, so we can just clean
		// the name rather than force the user to do so.
		labels = labels[:last]
	}
	slices.Reverse(labels)

	ret := Name{
		labels: make([]Label, 0, len(labels)),
	}
	for _, l := range labels {
		ret.labels = append(ret.labels, Label{l})
	}
	return ret, nil
}

// String returns the domain name in its canonicalized PSL string
// format.
func (d Name) String() string {
	var b strings.Builder
	for i := len(d.labels) - 1; i >= 0; i-- {
		b.WriteString(d.labels[i].String())
		if i != 0 {
			b.WriteByte('.')
		}
	}
	return b.String()
}

// ASCIIString returns the domain name in its canonicalized ASCII (aka
// "punycode") form.
func (d Name) ASCIIString() string {
	var b strings.Builder
	for i := len(d.labels) - 1; i >= 0; i-- {
		b.WriteString(d.labels[i].ASCIIString())
		if i != 0 {
			b.WriteByte('.')
		}
	}
	return b.String()
}

// Compare compares domain names. It returns -1 if d < e, +1 if d > e,
// and 0 if d == e.
//
// Compare returns 0 for domain names that are equal as defined in
// IDNA2008. Unequal domain names are ordered according to
// Label.Compare of their first unequal label, starting from the TLD.
func (d Name) Compare(e Name) int {
	return slices.CompareFunc(d.labels, e.labels, Label.Compare)
}

// Equal reports whether d and e are equal.
//
// Equality is as defined in IDNA2008.
func (d Name) Equal(e Name) bool { return d.Compare(e) == 0 }

// NumLabels returns the number of DNS labels in the domain name.
func (d Name) NumLabels() int { return len(d.labels) }

// Labels returns the individual labels of the domain name.
func (d Name) Labels() []Label {
	// Make a copy because we need to reverse the entries, and also we
	// have to maintain the invariant that Name is always in canonical
	// form, and allowing the caller to mutate the labels via this
	// slice would defeat that.
	ret := append([]Label(nil), d.labels...)
	slices.Reverse(ret)
	return ret
}

// CutSuffix removes suffix from d. If d is a child domain of suffix,
// CutSuffix returns the remaining leaf labels and found=true.
// Otherwise, it returns nil, false.
func (d Name) CutSuffix(suffix Name) (rest []Label, found bool) {
	// In DNS terms, a suffix must leave at least one non-suffix
	// label, so d == suffix fails the cut.
	if len(suffix.labels) >= len(d.labels) {
		return nil, false
	}

	cutIdx := len(suffix.labels)

	if !slices.EqualFunc(d.labels[:cutIdx], suffix.labels, Label.Equal) {
		return nil, false
	}

	ret := append([]Label(nil), d.labels[cutIdx:]...)
	slices.Reverse(ret)
	return ret, true
}

// AddPrefix returns d prefixed with labels.
//
// For example, AddPrefix("qux", "bar") to "foo.com" is "qux.bar.foo.com".
func (d Name) AddPrefix(labels ...Label) (Name, error) {
	// Due to total name length restrictions, we have to fully
	// re-check the shape of the extended domain name. The simplest
	// way to do that is to round-trip through a string and leverage
	// Parse again.
	parts := make([]string, 0, len(labels)+1)
	for _, l := range labels {
		parts = append(parts, l.String())
	}
	parts = append(parts, d.String())
	retStr := strings.Join(parts, ".")
	return Parse(retStr)
}

// MustAddPrefix is like AddPrefix, but panics if the formed prefix is
// invalid instead of returning an error.
func (d Name) MustAddPrefix(labels ...Label) Name {
	ret, err := d.AddPrefix(labels...)
	if err != nil {
		panic(fmt.Sprintf("failed to add prefix %v to domain %q: %v", labels, d, err))
	}
	return ret
}

// Label is a domain name label.
type Label struct {
	label string
}

// ParseLabel parses and validates a domain name label.
//
// s is validated and canonicalized into the Unicode form suitable for
// use in domain registrations, as defined by IDNA2008.
func ParseLabel(s string) (Label, error) {
	canonical, err := domainValidator.ToUnicode(s)
	if err != nil {
		return Label{}, err
	} else if strings.Contains(canonical, ".") {
		return Label{}, fmt.Errorf("label %q cannot contain a dot", s)
	}

	return Label{canonical}, nil
}

func (l Label) String() string { return l.label }

func (l Label) ASCIIString() string {
	ret, err := domainValidator.ToASCII(l.label)
	if err != nil {
		// This should be impossible. Domain labels can only be
		// created by ParseLabel, which applies IDNA validation and
		// produces a canonical U-label. We're just converting from
		// U-label representation to A-label, which is guaranteed to
		// succeed given a valid U-label.
		panic(fmt.Sprintf("impossible: U-label to A-label conversion failed: %v", err))
	}
	return ret
}

// AsTLD returns the label as a top-level domain Name.
func (l Label) AsTLD() Name {
	return Name{
		labels: []Label{l},
	}
}

// Compare compares domain labels. It returns -1 if l < m, +1 if l > m,
// and 0 if l == m.
//
// Compare returns 0 for labels that are equal as defined in
// IDNA2008. Unequal labels are ordered by lexical byte-wise
// comparison of their IDNA2008 canonical forms.
func (l Label) Compare(m Label) int {
	// IDNA2008 specifies that U-labels (which is what we store) can
	// by compared bytewise, as long as you respect some constraints:
	// the comparison must be case-insensitive, using the case mapping
	// defined by IDNA (NFKC_Casefold plus some IDNA-specific
	// mappings); and all inputs must use a consistent representation,
	// e.g. all UTF-8, or all UCS-2, but not mixed UTF-8 and UCS-2.
	//
	// x/net/idna implements the TR46 algorithm
	// (https://www.unicode.org/reports/tr46/), which among other
	// things applies IDNA case mapping to labels. So, by the time the
	// strings are in a DomainLabel, they have been preprocessed such
	// that builtin string comparison is correct.
	bytewiseCmp := cmp.Compare(l.label, m.label)
	if bytewiseCmp == 0 {
		return 0
	}

	// If two labels aren't equal, we are free to order them however
	// we want. We choose to order them with the English Unicode
	// collation.
	if res := compareLabel(l, m); res != 0 {
		return res
	}

	// labelCollator reported equivalent but not bit-identical
	// strings. To avoid violating IDNA's definition of equality,
	// break the tie using byte order.
	//
	// This is called "deterministic sorting" in TR46. Unicode
	// discourages its use for general purpose sorting, but in this
	// case it's the perfect tool, because it lets us get Unicode's
	// good sorting for things that are visually different to humans,
	// but also lets us comply with IDNA's definition of equality.
	return bytewiseCmp
}

// Equal reports whether domain labels are equal.
//
// Equality is as defined in IDNA2008.
func (l Label) Equal(m Label) bool { return l.Compare(m) == 0 }

// domainValidator is the IDNA profile used to parse, validate and
// canonicalize domain names in the PSL. It is equivalent to RFC
// 5891's strictest rules (the ones for domain registration), with one
// exception: inputs that are valid but non-canonical are rewritten to
// canonical form, rather than rejected.
//
// This deviation is a concession to usability, because the PSL is
// maintained by humans typing into text editors, and unless that
// editor is a hex editor it can be difficult to write Unicode exactly
// like IDNA wants.
//
// Instead of forcing humans to debug "your bytes are wrong" error for
// strings that look identical, we canonicalize inputs in the same way
// as a browser's URL bar, and then we apply the stricter-than-browser
// rules for domain registration.
//
// Note that the PSL's official format still requires suffixes in
// canonical IDNA form. This parser is intentionally less strict it
// wants to help contributors make their changes canonical before they
// send a PR.
var domainValidator = idna.New(
	// Map acceptable but non-canonical characters to their canonical
	// value, instead of rejecting the input entirely.
	idna.MapForLookup(),
	// Validate bidirectional text according to RFC 5893, section 2
	idna.BidiRule(),
	// Validate labels according to RFC 5891 section 5.4
	idna.ValidateLabels(true),
	// Strictly limit allowed ASCII characters according to RFC 1034
	// section 3.5. This matches the requirement for domain
	// registration under ICANN TLDs (but excludes things like _proto
	// SRV labels).
	idna.StrictDomainName(true),
	// Validate label and overall domain name length according to RFC
	// 1035 section 2.3.4.
	idna.VerifyDNSLength(true),
	// Use non-transitional IDNA2008 mapping, without IDNA2003
	// compatibility hacks. This matches the current behavior of all
	// major browsers, and transitional processing is officially
	// deprecated as of Unicode 15.1 (2023).
	idna.Transitional(false),
	// Do not remove leading dots. It would be nice to do this
	// cleanup, but TR46 doesn't include this processing step, and
	// thus some IDNA test vectors produce the wrong result. So we
	// have a choice: do we want to test this code using the official
	// Unicode test vectors, or do we want slightly better handling of
	// rare invalid input? We choose the better tests.
	idna.RemoveLeadingDots(false),
)

// IDNA/Unicode do not define a canonical order for unequal domain
// labels. The exact order we pick doesn't matter because nothing in
// the DNS/domain ecosystem cares what the relative order of domain
// names is. However, we would like the order to make sense to humans,
// especially for non-latin scripts where byte order is very wrong.
//
// Similar to the sibling parser package, we use the basic English
// collation, which produces the right result for English, and
// reasonably good results for other languages.
//
// Nothing except Label.Compare should use this collator. In
// particular, it MUST NOT be used by itself to establish equality of
// labels, because we have to obey IDNA's strict definition of
// equality.
//
// In theory, x/text/collate has the collate.Force option for exactly
// what we need: it breaks ties between equivalent inputs by doing a
// byte compare. However, this option is buggy and silently ignored in
// some cases (https://github.com/golang/go/issues/68379), so we do
// this tie breaking ourselves in Label.Compare.
var labelCollatorMu sync.Mutex
var labelCollator = collate.New(language.English)

func compareLabel(a, b Label) int {
	// Unfortunately individual collators are not safe for concurrent
	// use. Wrap them in a global mutex. We could also construct a new
	// collator for each use, but that ends up being more expensive
	// and less performant than sharing one collator with a mutex.
	labelCollatorMu.Lock()
	defer labelCollatorMu.Unlock()
	var buf collate.Buffer
	kl := labelCollator.KeyFromString(&buf, a.label)
	km := labelCollator.KeyFromString(&buf, b.label)
	return bytes.Compare(kl, km)
}
