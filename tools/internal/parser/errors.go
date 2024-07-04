package parser

import (
	"fmt"
	"strings"
)

// ErrInvalidEncoding reports that the input is encoded with
// something other than UTF-8.
type ErrInvalidEncoding struct {
	Encoding string
}

func (e ErrInvalidEncoding) Error() string {
	return fmt.Sprintf("invalid character encoding %s", e.Encoding)
}

// ErrUTF8BOM reports that the input has an unnecessary UTF-8 byte
// order mark (BOM) at the start.
type ErrUTF8BOM struct{}

func (e ErrUTF8BOM) Error() string { return "file has a UTF-8 byte order mark (BOM)" }

// ErrInvalidUTF8 reports that a line contains bytes that are not
// valid UTF-8.
type ErrInvalidUTF8 struct {
	SourceRange
}

func (e ErrInvalidUTF8) Error() string {
	return fmt.Sprintf("%s: invalid UTF-8 bytes", e.SourceRange.LocationString())
}

// ErrDOSNewline reports that a line has a DOS style line ending.
type ErrDOSNewline struct {
	SourceRange
}

func (e ErrDOSNewline) Error() string {
	return fmt.Sprintf("%s: found DOS line ending (\\r\\n instead of just \\n)", e.SourceRange.LocationString())
}

// ErrTrailingWhitespace reports that a line has trailing whitespace.
type ErrTrailingWhitespace struct {
	SourceRange
}

func (e ErrTrailingWhitespace) Error() string {
	return fmt.Sprintf("%s: trailing whitespace", e.SourceRange.LocationString())
}

// ErrLeadingWhitespace reports that a line has leading whitespace.
type ErrLeadingWhitespace struct {
	SourceRange
}

func (e ErrLeadingWhitespace) Error() string {
	return fmt.Sprintf("%s: leading whitespace", e.SourceRange.LocationString())
}

// ErrSectionInSuffixBlock reports that a comment within a suffix
// block contains a section delimiter.
type ErrSectionInSuffixBlock struct {
	SourceRange
}

func (e ErrSectionInSuffixBlock) Error() string {
	return fmt.Sprintf("%s: section delimiter not allowed in suffix block comment", e.SourceRange.LocationString())
}

// ErrUnclosedSection reports that a file section was not closed
// properly before EOF.
type ErrUnclosedSection struct {
	Section *Section
}

func (e ErrUnclosedSection) Error() string {
	return fmt.Sprintf("%s: section %q is missing its closing marker", e.Section.SourceRange.LocationString(), e.Section.Name)
}

// ErrNestedSection reports that a file section is being started while
// already within a section.
type ErrNestedSection struct {
	SourceRange
	Name    string
	Section *Section
}

func (e ErrNestedSection) Error() string {
	return fmt.Sprintf("%s: section %q is nested inside section %q (%s)", e.SourceRange.LocationString(), e.Name, e.Section.Name, e.Section.SourceRange.LocationString())
}

// ErrUnstartedSection reports that section end marker was found
// without a corresponding start.
type ErrUnstartedSection struct {
	SourceRange
	Name string
}

func (e ErrUnstartedSection) Error() string {
	return fmt.Sprintf("%s: end marker for non-existent section %q", e.SourceRange.LocationString(), e.Name)
}

// ErrMismatchedSection reports that a file section was started
// under one name but ended under another.
type ErrMismatchedSection struct {
	SourceRange
	EndName string
	Section *Section
}

func (e ErrMismatchedSection) Error() string {
	return fmt.Sprintf("%s: section %q (%s) closed with wrong name %q", e.SourceRange.LocationString(), e.Section.Name, e.Section.SourceRange.LocationString(), e.EndName)
}

// ErrUnknownSectionMarker reports that a line looks like a file section
// marker (e.g. "===BEGIN ICANN DOMAINS==="), but is not one of the
// recognized kinds of marker.
type ErrUnknownSectionMarker struct {
	SourceRange
}

func (e ErrUnknownSectionMarker) Error() string {
	return fmt.Sprintf("%s: unknown kind of section marker", e.SourceRange.LocationString())
}

// MissingEntityName reports that a block of suffixes does not have a
// parseable owner name in its header comment.
type ErrMissingEntityName struct {
	Suffixes *Suffixes
}

func (e ErrMissingEntityName) Error() string {
	return fmt.Sprintf("%s: suffix block has no owner name", e.Suffixes.SourceRange.LocationString())
}

// ErrMissingEntityEmail reports that a block of suffixes does not have a
// parseable contact email address in its header comment.
type ErrMissingEntityEmail struct {
	Suffixes *Suffixes
}

func (e ErrMissingEntityEmail) Error() string {
	return fmt.Sprintf("%s: suffix block has no contact email", e.Suffixes.SourceRange.LocationString())
}

// ErrSuffixBlocksInWrongPlace reports that some suffix blocks of the
// private section are in the wrong sort order.
type ErrSuffixBlocksInWrongPlace struct {
	// EditScript is a list of suffix block movements to put the
	// private domains section in the correct order. Note that each
	// step assumes that the previous steps have already been done.
	EditScript []MoveSuffixBlock
}

// MoveSuffixBlock describes the movement of one suffix block to a
// different place in the PSL file.
type MoveSuffixBlock struct {
	// Name is the name of the block to be moved.
	Name string
	// InsertAfter is the name of the block that is immediately before
	// the correct place to insert Block, or the empty string if Block
	// should go first in the private domains section.
	InsertAfter string
}

func (e ErrSuffixBlocksInWrongPlace) Error() string {
	if len(e.EditScript) == 1 {
		after := e.EditScript[0].InsertAfter
		if after == "" {
			return fmt.Sprintf("suffix block %q is in the wrong place, should be at the start of the private section", e.EditScript[0].Name)
		} else {
			return fmt.Sprintf("suffix block %q is in the wrong place, it should go immediately after block %q", e.EditScript[0].Name, e.EditScript[0].InsertAfter)
		}
	}

	var ret strings.Builder
	fmt.Fprintf(&ret, "%d suffix blocks are in the wrong place, make these changes to fix:\n", len(e.EditScript))

	for _, edit := range e.EditScript {
		fmt.Fprintf(&ret, "\tmove block: %s\n", edit.Name)
		if edit.InsertAfter == "" {
			fmt.Fprintf(&ret, "\t        to: start of private section\n")
		} else {
			fmt.Fprintf(&ret, "\t     after: %s\n", edit.InsertAfter)
		}
	}

	return ret.String()
}

// ErrInvalidSuffix reports that a suffix suffix is not a valid PSL
// entry.
type ErrInvalidSuffix struct {
	SourceRange
	Suffix string
	Err    error
}

func (e ErrInvalidSuffix) Error() string {
	return fmt.Sprintf("%s: invalid suffix %q: %v", e.SourceRange.LocationString(), e.Suffix, e.Err)
}
