package parser

import (
	"fmt"
	"strings"
)

// InvalidEncodingError reports that the input is encoded with
// something other than UTF-8.
type InvalidEncodingError struct {
	Encoding string
}

func (e InvalidEncodingError) Error() string {
	return fmt.Sprintf("file uses invalid character encoding %s", e.Encoding)
}

// UTF8BOMError reports that the input has an unnecessary UTF-8 byte
// order mark (BOM) at the start.
type UTF8BOMError struct{}

func (e UTF8BOMError) Error() string {
	return "file starts with an unnecessary UTF-8 BOM (byte order mark)"
}

// InvalidUTF8Error reports that a line contains bytes that are not
// valid UTF-8.
type InvalidUTF8Error struct {
	Line Source
}

func (e InvalidUTF8Error) Error() string {
	return fmt.Sprintf("found non UTF-8 bytes at %s", e.Line.LocationString())
}

// DOSNewlineError reports that a line has a DOS style line ending.
type DOSNewlineError struct {
	Line Source
}

func (e DOSNewlineError) Error() string {
	return fmt.Sprintf("%s has a DOS line ending (\\r\\n instead of just \\n)", e.Line.LocationString())
}

// TrailingWhitespaceError reports that a line has trailing whitespace.
type TrailingWhitespaceError struct {
	Line Source
}

func (e TrailingWhitespaceError) Error() string {
	return fmt.Sprintf("%s has trailing whitespace", e.Line.LocationString())
}

// LeadingWhitespaceError reports that a line has leading whitespace.
type LeadingWhitespaceError struct {
	Line Source
}

func (e LeadingWhitespaceError) Error() string {
	return fmt.Sprintf("%s has leading whitespace", e.Line.LocationString())
}

// SectionInSuffixBlock reports that a comment within a block of
// suffixes contains a section delimiter.
type SectionInSuffixBlock struct {
	Line Source
}

func (e SectionInSuffixBlock) Error() string {
	return fmt.Sprintf("section delimiters are not allowed in suffix block comment at %s", e.Line.LocationString())
}

// UnclosedSectionError reports that a file section was not closed
// properly before EOF.
type UnclosedSectionError struct {
	Start *StartSection // The unpaired section start
}

func (e UnclosedSectionError) Error() string {
	return fmt.Sprintf("section %q started at %s, but is never closed", e.Start.Name, e.Start.LocationString())
}

// NestedSectionError reports that a file section is being started
// while already within a section, which the PSL format does not
// allow.
type NestedSectionError struct {
	Outer *StartSection
	Inner *StartSection
}

func (e NestedSectionError) Error() string {
	return fmt.Sprintf("new section %q started at %s while still in section %q (started at %s)", e.Inner.Name, e.Inner.LocationString(), e.Outer.Name, e.Outer.LocationString())
}

// UnstartedSectionError reports that a file section end marker was
// found without a corresponding start.
type UnstartedSectionError struct {
	End *EndSection
}

func (e UnstartedSectionError) Error() string {
	return fmt.Sprintf("section %q closed at %s but was not started", e.End.Name, e.End.LocationString())
}

// MismatchedSectionError reports that a file section was started
// under one name but ended under another.
type MismatchedSectionError struct {
	Start *StartSection
	End   *EndSection
}

func (e MismatchedSectionError) Error() string {
	return fmt.Sprintf("section %q closed at %s while in section %q (started at %s)", e.End.Name, e.End.LocationString(), e.Start.Name, e.Start.LocationString())
}

// UnknownSectionMarker reports that a line looks like a file section
// marker (e.g. "===BEGIN ICANN DOMAINS==="), but is not one of the
// recognized kinds of marker.
type UnknownSectionMarker struct {
	Line Source
}

func (e UnknownSectionMarker) Error() string {
	return fmt.Sprintf("unknown kind of section marker %q at %s", e.Line.Text(), e.Line.LocationString())
}

// UnterminatedSectionMarker reports that a section marker is missing
// the required trailing "===", e.g. "===BEGIN ICANN DOMAINS".
type UnterminatedSectionMarker struct {
	Line Source
}

func (e UnterminatedSectionMarker) Error() string {
	return fmt.Sprintf(`section marker %q at %s is missing trailing "==="`, e.Line.Text(), e.Line.LocationString())
}

// MissingEntityName reports that a block of suffixes does not have a
// parseable owner name in its header comment.
type MissingEntityName struct {
	Suffixes *Suffixes
}

func (e MissingEntityName) Error() string {
	return fmt.Sprintf("could not find entity name for %s at %s", e.Suffixes.shortName(), e.Suffixes.LocationString())
}

// MissingEntityEmail reports that a block of suffixes does not have a
// parseable contact email address in its header comment.
type MissingEntityEmail struct {
	Suffixes *Suffixes
}

func (e MissingEntityEmail) Error() string {
	return fmt.Sprintf("could not find a contact email for %s at %s", e.Suffixes.shortName(), e.Suffixes.LocationString())
}

// SuffixBlocksInWrongPlace reports that some suffix blocks of the
// private section are in the wrong sort order.
type SuffixBlocksInWrongPlace struct {
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

func (e SuffixBlocksInWrongPlace) Error() string {
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
