package parser

import (
	"cmp"
	"fmt"
	"net/mail"
	"net/url"
	"slices"

	"github.com/publicsuffix/list/tools/internal/domain"
)

// A Block is a parsed chunk of a PSL file. Each block is one of the
// concrete types Comment, Section, Suffixes, Suffix, or Wildcard.
type Block interface {
	// SrcRange returns the block's SourceRange.
	SrcRange() SourceRange
	// Children returns the block's direct children, if any.
	Children() []Block
	// Changed reports whether the tree rooted at block has changed
	// since the base of comparison (see List.SetBaseVersion).
	Changed() bool

	info() *blockInfo
}

// BlocksOfType recursively collects and returns all blocks of
// concrete type T in the given parse tree.
//
// For example, BlocksOfType[*parser.Comment](ast) returns all comment
// nodes in ast.
func BlocksOfType[T Block](tree Block) []T {
	var ret []T
	blocksOfTypeRec(tree, &ret)
	return ret
}

func blocksOfTypeRec[T Block](tree Block, out *[]T) {
	if v, ok := tree.(T); ok {
		*out = append(*out, v)
	}
	for _, child := range tree.Children() {
		blocksOfTypeRec(child, out)
	}
}

// blockInfo is common information shared by all Block types.
type blockInfo struct {
	SourceRange

	// isUnchanged records that a Block (including any children) is
	// semantically unchanged from a past base point. The default base
	// of comparison is a null List, meaning that Unchanged=false for
	// all blocks. A different base of comparison can be set with
	// List.Diff.
	isUnchanged bool
}

func (b blockInfo) SrcRange() SourceRange {
	return b.SourceRange
}

func (b blockInfo) Changed() bool {
	return !b.isUnchanged
}

func (b *blockInfo) info() *blockInfo {
	return b
}

// List is a parsed public suffix list.
type List struct {
	blockInfo

	// Blocks are the top-level elements of the list, in the order
	// they appear.
	Blocks []Block
}

func (l *List) Children() []Block { return l.Blocks }

// PublicSuffix returns the public suffix of n.
//
// This follows the PSL algorithm to the letter. Notably: a rule
// "*.foo.com" does not implicitly create a "foo.com" rule, and there
// is a hardcoded implicit "*" rule so that unknown TLDs are all
// public suffixes.
func (l *List) PublicSuffix(d domain.Name) domain.Name {
	if d.NumLabels() == 0 {
		// Edge case: zero domain.Name value
		return d
	}

	// Look at wildcards first, because the PSL algorithm says that
	// exceptions to wildcards take priority over all other rules. So,
	// if we find a wildcard exception, we can halt early.
	var (
		ret          domain.Name
		matchLen     int
		gotException bool
	)
	for _, w := range BlocksOfType[*Wildcard](l) {
		suf, isException, ok := w.PublicSuffix(d)
		switch {
		case !ok:
			continue
		case isException && !gotException:
			// First matching exception encountered.
			gotException = true
			matchLen = suf.NumLabels()
			ret = suf
		case isException:
			// Second or later exception match. According to the
			// format, only 0 or 1 exceptions can match,
			// multi-exception matches are undefined and unused. But
			// just to be safe, handle the N exception case by
			// accepting the longest matching exception.
			if nl := suf.NumLabels(); nl > matchLen {
				matchLen = nl
				ret = suf
			}
		case !gotException:
			// Non-exception match.
			if nl := suf.NumLabels(); nl > matchLen {
				matchLen = nl
				ret = suf
			}
		}
	}
	if gotException {
		return ret
	}

	// Otherwise, keep scanning through the regular suffixes.
	for _, s := range BlocksOfType[*Suffix](l) {
		if suf, ok := s.PublicSuffix(d); ok && suf.NumLabels() > matchLen {
			matchLen = suf.NumLabels()
			ret = suf
		}
	}

	if matchLen == 0 {
		// The PSL algorithm includes an implicit "*" to match every
		// TLD, in the absence of any matching explicit rule.
		labels := d.Labels()
		tld := labels[len(labels)-1].AsTLD()
		return tld
	}

	return ret
}

// RegisteredDomain returns the registered/registerable domain of
// n. Returns (domain, true) when the input is a child of a public
// suffix, and (zero, false) when the input is itself a public suffix.
//
// RegisteredDomain follows the PSL algorithm to the letter. Notably:
// a rule "*.foo.com" does not implicitly create a "foo.com" rule, and
// there is a hardcoded implicit "*" rule so that unknown TLDs are all
// public suffixes.
func (l *List) RegisteredDomain(d domain.Name) (domain.Name, bool) {
	suf := l.PublicSuffix(d)
	if suf.Equal(d) {
		return domain.Name{}, false
	}

	next, ok := d.CutSuffix(suf)
	if !ok {
		panic(fmt.Sprintf("public suffix %q is not a suffix of domain %q", suf, d))
	}
	return suf.MustAddPrefix(next[len(next)-1]), true
}

// Comment is a comment block, consisting of one or more contiguous
// lines of commented text.
type Comment struct {
	blockInfo
	// Text is the unprocessed content of the comment lines, with the
	// leading comment syntax removed.
	Text []string
}

func (c *Comment) Children() []Block { return nil }

// Section is a named part of a PSL file, containing suffixes which
// behave similarly.
type Section struct {
	blockInfo

	// Name is he section name. In a normal well-formed PSL file, the
	// names are "ICANN DOMAINS" and "PRIVATE DOMAINS".
	Name string
	// Blocks are the child blocks contained within the section.
	Blocks []Block
}

func (s *Section) Children() []Block { return s.Blocks }

// Suffixes is a list of PSL domain suffixes with optional additional
// metadata.
//
// Suffix sections consist of a header comment that contains a mix of
// structured and unstructured information, followed by a list of
// domain suffixes. The suffix list may contain additional
// unstructured inline comments.
type Suffixes struct {
	blockInfo

	// Info is information about the authoritative maintainers for
	// this set of suffixes.
	Info MaintainerInfo

	// Blocks are the child blocks contained within the section.
	Blocks []Block
}

func (s *Suffixes) Children() []Block { return s.Blocks }

type MaintainerInfo struct {
	// Name is the name of the entity responsible for maintaining a
	// set of suffixes.
	//
	// For ICANN suffixes, this is typically the TLD name, or the name
	// of NIC that controls the TLD.
	//
	// For private domains this is the name of the legal entity
	// (usually a company, sometimes an individual) that owns all
	// domains in the block.
	//
	// In a well-formed PSL file, Name is non-empty for all suffix
	// blocks.
	Name string

	// URLs are links to further information about the suffix block's
	// domains and its maintainer.
	//
	// For ICANN domains this is typically the NIC's information page
	// for the TLD, or failing that a general information page such as
	// a Wikipedia entry.
	//
	// For private domains this is usually the website for the owner
	// of the domains.
	//
	// May be empty when the block header doesn't have
	// machine-readable URLs.
	URLs []*url.URL

	// Maintainer is the contact name and email address of the person
	// or persons responsible for maintaining a block.
	//
	// This field may be empty if there is no machine-readable contact
	// information.
	Maintainers []*mail.Address

	// Other is some unstructured additional notes. They may contain
	// anything, including some of the above information that wasn't
	// in a known parseable form.
	Other []string

	// MachineEditable is whether this information can be
	// machine-edited and written back out without loss of
	// information. The exact formatting of the information may
	// change, but no information will be lost.
	MachineEditable bool
}

func (m *MaintainerInfo) Compare(n *MaintainerInfo) int {
	if r := compareCommentText(m.Name, n.Name); r != 0 {
		return r
	}

	if r := cmp.Compare(len(m.URLs), len(n.URLs)); r != 0 {
		return r
	}
	for i := range m.URLs {
		if r := cmp.Compare(m.URLs[i].String(), n.URLs[i].String()); r != 0 {
			return r
		}
	}

	if r := cmp.Compare(len(m.Maintainers), len(n.Maintainers)); r != 0 {
		return r
	}
	for i := range m.Maintainers {
		if r := cmp.Compare(m.Maintainers[i].String(), n.Maintainers[i].String()); r != 0 {
			return r
		}
	}

	if r := slices.Compare(m.Other, n.Other); r != 0 {
		return r
	}

	if m.MachineEditable == n.MachineEditable {
		return 0
	} else if !m.MachineEditable {
		return -1
	} else {
		return 1
	}
}

// HasInfo reports whether m has any maintainer information at all.
func (m MaintainerInfo) HasInfo() bool {
	return m.Name != "" || len(m.URLs) > 0 || len(m.Maintainers) > 0 || len(m.Other) > 0
}

// Suffix is one public suffix, represented in the standard domain
// name format.
type Suffix struct {
	blockInfo

	// Domain is the public suffix's domain name.
	Domain domain.Name
}

func (s *Suffix) Children() []Block { return nil }

// PublicSuffix returns the public suffix of n according to this
// Suffix rule taken in isolation. If n is not a child domain of s
// PublicSuffix returns (zeroValue, false).
func (s *Suffix) PublicSuffix(n domain.Name) (suffix domain.Name, ok bool) {
	if n.Equal(s.Domain) {
		return s.Domain, true
	}
	if _, ok := n.CutSuffix(s.Domain); ok {
		return s.Domain, true
	}
	return domain.Name{}, false
}

// RegisteredDomain returns the registered/registerable domain of n
// according to this Suffix rule taken in isolation. The registered
// domain is defined as n's public suffix plus one more child
// label. If n is not a child domain of s, RegisteredDomain returns
// (zeroValue, false).
func (s *Suffix) RegisteredDomain(n domain.Name) (regDomain domain.Name, ok bool) {
	if prefix, ok := n.CutSuffix(s.Domain); ok {
		return s.Domain.MustAddPrefix(prefix[len(prefix)-1]), true
	}
	return domain.Name{}, false
}

// Wildcard is a wildcard public suffix, along with any exceptions to
// that wildcard.
type Wildcard struct {
	blockInfo

	// Domain is the base of the wildcard public suffix, without the
	// leading "*" label.
	Domain domain.Name
	// Exceptions are the domain.Labels that, when they appear in the
	// wildcard position of Domain, cause a FQDN to _not_ match this
	// wildcard. For example, if Domain="foo.com" and Exceptions=[bar,
	// qux], zot.foo.com is a public suffix, but bar.foo.com and
	// qux.foo.com are not.
	Exceptions []domain.Label
}

func (w *Wildcard) Children() []Block { return nil }

// PublicSuffix returns the public suffix of n according to this
// Wildcard rule taken in isolation. If n is not a child domain of w
// PublicSuffix returns (zeroValue, false).
func (w *Wildcard) PublicSuffix(n domain.Name) (suffix domain.Name, isException, ok bool) {
	if prefix, ok := n.CutSuffix(w.Domain); ok {
		next := prefix[len(prefix)-1]
		if slices.Contains(w.Exceptions, next) {
			return w.Domain, true, true
		}

		return w.Domain.MustAddPrefix(next), false, true
	}
	return domain.Name{}, false, false
}

// RegisteredDomain returns the registered/registerable domain of n
// according to this Suffix rule taken in isolation. The registered
// domain is defined as n's public suffix plus one more child
// label. If n is not a child domain of s, RegisteredDomain returns
// (zeroValue, false).
func (w *Wildcard) RegisteredDomain(n domain.Name) (regDomain domain.Name, isException, ok bool) {
	if prefix, ok := n.CutSuffix(w.Domain); ok && len(prefix) >= 2 {
		next := prefix[len(prefix)-1]
		if slices.Contains(w.Exceptions, next) {
			return w.Domain.MustAddPrefix(next), true, true
		}

		return w.Domain.MustAddPrefix(prefix[len(prefix)-2:]...), false, true
	}
	return domain.Name{}, false, false
}
