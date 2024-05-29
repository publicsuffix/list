package parser

import (
	"fmt"
	"net/mail"
	"net/url"
)

// File is a parsed PSL file.
// A PSL file consists of blocks separated by an empty line. Most
// blocks are annotated lists of suffixes, but some are plain
// top-level comments or delimiters for sections of the file.
type File struct {
	// Blocks are the data blocks of the file, in the order they
	// appear.
	Blocks []Block
	// Errors are parse errors encountered while reading the
	// file. This includes fatal validation errors, not just malformed
	// syntax.
	Errors []error
	// Warnings are errors that were downgraded to just
	// warnings. Warnings are a concession to old PSL entries that now
	// have validation errors, due to PSL policy changes. As long as
	// the entries in question don't change, their preexisting
	// validation errors are downgraded to lint warnings.
	Warnings []error
}

// AllSuffixBlocks returns all suffix blocks in f.
func (f *File) AllSuffixBlocks() []Suffixes {
	var ret []Suffixes

	for _, block := range f.Blocks {
		switch v := block.(type) {
		case Suffixes:
			ret = append(ret, v)
		}
	}

	return ret
}

// SuffixBlocksInSection returns all suffix blocks within the named
// file section (for example, "ICANN DOMAINS" or "PRIVATE DOMAINS").
func (f *File) SuffixBlocksInSection(name string) []Suffixes {
	var ret []Suffixes

	var curSection string
	for _, block := range f.Blocks {
		switch v := block.(type) {
		case StartSection:
			curSection = v.Name
		case EndSection:
			if curSection == name {
				return ret
			}
			curSection = ""
		case Suffixes:
			if curSection == name {
				ret = append(ret, v)
			}
		}
	}
	return ret
}

// Source is a piece of source text with location information.
type Source struct {
	// StartLine is the first line of this piece of source text in the
	// original file. The first line of a file is line 1 rather than
	// line 0, since that is how text editors conventionally number
	// lines.
	StartLine int
	// EndLine is the last line of this piece of source text in the
	// original file. The line named by EndLine is included in the
	// source block.
	EndLine int
	// Raw is the unparsed source text for this block.
	Raw string
}

// LocationString returns a short string describing the source
// location.
func (s Source) LocationString() string {
	if s.StartLine == s.EndLine {
		return fmt.Sprintf("line %d", s.StartLine)
	}
	return fmt.Sprintf("lines %d-%d", s.StartLine, s.EndLine)
}

// A Block is a parsed chunk of a PSL file.
// In Parse's output, a Block is one of the following concrete types:
// Comment, StartSection, EndSection, Suffixes.
type Block interface {
	source() Source
}

// Comment is a standalone top-level comment block.
type Comment struct {
	Source
}

func (c Comment) source() Source { return c.Source }

// StartSection is a top-level marker that indicates the start of a
// logical section, such as ICANN suffixes or privately managed
// domains.
//
// Sections cannot be nested, at any one point in a file you are
// either not in any logical section, or within a single section.  In
// a File that has no parse errors, StartSection and EndSection blocks
// are correctly paired, and all sections are closed by an EndSection
// before any following StartSection.
type StartSection struct {
	Source
	Name string // section name, e.g. "ICANN DOMAINS", "PRIVATE DOMAINS"
}

func (b StartSection) source() Source { return b.Source }

// EndSection is a top-level marker that indicates the end of a
// logical section, such as ICANN suffixes or privately managed
// domains.
//
// Sections cannot be nested, at any one point in a file you are
// either not in any logical section, or within a single section.  In
// a File that has no parse errors, StartSection and EndSection blocks
// are correctly paired, and all sections are closed by an EndSection
// before any following StartSection.
type EndSection struct {
	Source
	Name string // e.g. "ICANN DOMAINS", "PRIVATE DOMAINS"
}

func (b EndSection) source() Source { return b.Source }

// Suffixes is a list of PSL domain suffixes with optional additional
// metadata.
//
// Suffix sections consist of a header comment that contains a mix of
// structured and unstructured information, followed by a list of
// domain suffixes. The suffix list may contain additional
// unstructured inline comments.
type Suffixes struct {
	Source

	// Header lists the comment lines that appear before the first
	// domain suffix. Any structured data they contain is also parsed
	// into separate fields.
	Header []Source
	// Entries lists the lines that contain domain suffixes. In an
	// error-free PSL file, each slice element is a single suffix.
	Entries []Source
	// InlineComments lists the comment lines that appear between
	// suffix lines, rather than as part of the header. These are
	// uncommon in the PSL overall, but some suffix blocks
	// (particularly hand-curated ICANN blocks) feature some guidance
	// comments to guide future maintainers.
	InlineComments []Source

	// The following fields are extracted from Header, if available.

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
}

func (s Suffixes) source() Source { return s.Source }

// shortName returns either the quoted name of the responsible Entity,
// or a generic descriptor of this suffix block if Entity is unset.
func (s Suffixes) shortName() string {
	if s.Entity != "" {
		return fmt.Sprintf("%q", s.Entity)
	}
	return fmt.Sprintf("%d unowned suffixes", len(s.Entries))
}
