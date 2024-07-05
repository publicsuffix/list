package parser

import (
	"cmp"
	"net/mail"
	"net/url"
	"slices"

	"github.com/publicsuffix/list/tools/internal/domain"
)

// List is a parsed public suffix list.
type List struct {
	SourceRange

	// Blocks are the top-level elements of the list, in the order
	// they appear.
	Blocks []Block
}

func (l *List) Children() []Block { return l.Blocks }

// A Block is a parsed chunk of a PSL file. Each block is one of the
// concrete types Comment, Section, Suffixes, Suffix, or Wildcard.
type Block interface {
	// SrcRange returns the block's SourceRange.
	SrcRange() SourceRange
	// Children returns the block's direct children, if any.
	Children() []Block
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

// Comment is a comment block, consisting of one or more contiguous
// lines of commented text.
type Comment struct {
	SourceRange
	// Text is the unprocessed content of the comment lines, with the
	// leading comment syntax removed.
	Text []string
}

func (c *Comment) Children() []Block { return nil }

// Section is a named part of a PSL file, containing suffixes which
// behave similarly.
type Section struct {
	SourceRange

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
	SourceRange

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

// Suffix is one public suffix, represented in the standard domain
// name format.
type Suffix struct {
	SourceRange

	// Domain is the public suffix's domain name.
	Domain domain.Name
}

func (s *Suffix) Children() []Block { return nil }

// Wildcard is a wildcard public suffix, along with any exceptions to
// that wildcard.
type Wildcard struct {
	SourceRange

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
