package parser

import (
	"fmt"
)

// UnclosedSectionError reports that a file section was not closed
// properly before EOF.
type UnclosedSectionError struct {
	Start StartSection // The unpaired section start
}

func (e UnclosedSectionError) Error() string {
	return fmt.Sprintf("section %q started at %s, but is never closed", e.Start.Name, e.Start.LocationString())
}

// NestedSectionError reports that a file section is being started
// while already within a section, which the PSL format does not
// allow.
type NestedSectionError struct {
	Outer StartSection
	Inner StartSection
}

func (e NestedSectionError) Error() string {
	return fmt.Sprintf("new section %q started at %s while still in section %q (started at %s)", e.Inner.Name, e.Inner.LocationString(), e.Outer.Name, e.Outer.LocationString())
}

// UnstartedSectionError reports that a file section end marker was
// found without a corresponding start.
type UnstartedSectionError struct {
	End EndSection
}

func (e UnstartedSectionError) Error() string {
	return fmt.Sprintf("section %q closed at %s but was not started", e.End.Name, e.End.LocationString())
}

// MismatchedSectionError reports that a file section was started
// under one name but ended under another.
type MismatchedSectionError struct {
	Start StartSection
	End   EndSection
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
	return fmt.Sprintf("unknown kind of section marker %q at %s", trimComment(e.Line.Raw), e.Line.LocationString())
}

// MissingEntityName reports that a block of suffixes does not have a
// parseable owner name in its header comment.
type MissingEntityName struct {
	Suffixes Suffixes
}

func (e MissingEntityName) Error() string {
	return fmt.Sprintf("could not find entity name for %s at %s", e.Suffixes.shortName(), e.Suffixes.LocationString())
}

// MissingEntityEmail reports that a block of suffixes does not have a
// parseable contact email address in its header comment.
type MissingEntityEmail struct {
	Suffixes Suffixes
}

func (e MissingEntityEmail) Error() string {
	return fmt.Sprintf("could not find a contact email for %s at %s", e.Suffixes.shortName(), e.Suffixes.LocationString())
}
