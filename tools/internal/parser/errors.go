package parser

import (
	"fmt"
)

// ErrInvalidEncoding reports that the input is encoded with
// something other than UTF-8.
type ErrInvalidEncoding struct {
	Encoding string
}

func (e ErrInvalidEncoding) Error() string {
	return fmt.Sprintf("invalid character encoding %s", e.Encoding)
}

// ErrInvalidUnicode reports that a line contains characters that are
// not valid Unicode.
type ErrInvalidUnicode struct {
	SourceRange
}

func (e ErrInvalidUnicode) Error() string {
	return fmt.Sprintf("%s: invalid Unicode character(s)", e.SourceRange.LocationString())
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

type ErrCommentPreventsSuffixSort struct {
	SourceRange
}

func (e ErrCommentPreventsSuffixSort) Error() string {
	return fmt.Sprintf("%s: comment prevents full sorting of suffixes", e.SourceRange.LocationString())
}

type ErrCommentPreventsSectionSort struct {
	SourceRange
}

func (e ErrCommentPreventsSectionSort) Error() string {
	return fmt.Sprintf("%s: comment prevents full sorting of PSL section", e.SourceRange.LocationString())
}
