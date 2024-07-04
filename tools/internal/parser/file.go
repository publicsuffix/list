package parser

import (
	"net/mail"
	"net/url"
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
// concrete types Blank, Comment, Section, Suffixes, Suffix, or
// Wildcard.
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

// Blank is a set of one or more consecutive blank lines.
type Blank struct {
	SourceRange
}

func (b *Blank) Children() []Block { return nil }

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
	// Entity is the name of the entity responsible for this block of
	// suffixes.
	//
	// For ICANN suffixes, this is typically the TLD name or the NIC
	// that controls the TLD.
	//
	// For private domains this is the name of the legal entity (most
	// commonly a company) that owns all domains in the block.
	//
	// In a well-formed PSL file, Entity is non-empty for all suffix
	// blocks.
	Entity string
	// URL is a link to further information about the suffix block and
	// its managing entity.
	//
	// For ICANN domains this is typically the NIC's information page
	// for the TLD, or failing that a general information page such as
	// a Wikipedia entry.
	//
	// For private domains this is usually the responsible company's
	// website.
	//
	// May be nil when the block header doesn't have a URL.
	URL *url.URL
	// Submitter is the contact name and email address of the person
	// or people responsible for this block of suffixes.
	//
	// This field may be nil if the block header doesn't have email
	// contact information.
	Submitter *mail.Address

	// Blocks are the child blocks contained within the section.
	Blocks []Block
}

func (s *Suffixes) Children() []Block { return s.Blocks }

// Suffix is one public suffix, represented in the standard domain
// name format.
type Suffix struct {
	SourceRange

	// Labels are the DNS labels of the public suffix.
	Labels []string
}

func (s *Suffix) Children() []Block { return nil }

// Wildcard is a wildcard public suffix, along with any exceptions to
// that wildcard.
type Wildcard struct {
	SourceRange

	// Labels are the DNS labels of the public suffix, without the
	// leading "*" label.
	Labels []string
	// Exceptions are the DNS label values that, when they appear in
	// the wildcard position, cause a FQDN to _not_ match this
	// wildcard. For example, if Labels=[foo, com] and
	// Exceptions=[bar, qux], zot.foo.com is a public suffix, but
	// bar.foo.com and qux.foo.com are not.
	Exceptions []string
}

func (w *Wildcard) Children() []Block { return nil }
