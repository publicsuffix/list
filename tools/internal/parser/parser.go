// Package parser implements a validating parser for the PSL files.
package parser

import (
	"strings"
)

// Parse parses bs as a PSL file and returns the parse result.
//
// The parser tries to keep going when it encounters errors. Parse and
// validation errors are accumulated in the Errors field of the
// returned File.
//
// If the returned File has a non-empty Errors field, the parsed file
// does not comply with the PSL format (documented at
// https://github.com/publicsuffix/list/wiki/Format), or with PSL
// submission guidelines
// (https://github.com/publicsuffix/list/wiki/Guidelines). A File with
// errors should not be used to calculate public suffixes for FQDNs.
func Parse(bs []byte) *File {
	return &parseWithExceptions(bs, downgradeToWarning, true).File
}

func parseWithExceptions(bs []byte, downgradeToWarning func(error) bool, validate bool) *parser {
	src, errs := newSource(bs)
	p := parser{
		downgradeToWarning: downgradeToWarning,
	}
	for _, err := range errs {
		p.addError(err)
	}
	p.Parse(src)
	if validate {
		p.Validate()
	}
	return &p
}

// parser is the state for a single PSL file parse.
type parser struct {
	// currentSection is the logical file section the parser is
	// currently in. This is used to verify that StartSection and
	// EndSection blocks are paired correctly, and may be nil when the
	// parser is not currently within a logical section.
	currentSection *StartSection

	// downgradeToWarning is a function that reports whether an error
	// should be recorded as a non-fatal warning. See exceptions.go
	// for the normal implementation. It's a struct field so that
	// tests can replace the normal list of exceptions with something
	// else for testing.
	downgradeToWarning func(error) bool

	// File is the parser's output.
	File
}

// Parse parses src as a PSL file and returns the parse result.
func (p *parser) Parse(src Source) {
	blankLine := func(line Source) bool { return line.Text() == "" }
	blocks := src.split(blankLine)

	for _, block := range blocks {
		// Does this block have any non-comments in it? If so, it's a
		// suffix block, otherwise it's a comment/section marker
		// block.
		notComment := func(line Source) bool { return !strings.HasPrefix(line.Text(), "//") }
		comment, rest, hasSuffixes := block.cut(notComment)
		if hasSuffixes {
			p.processSuffixes(block, comment, rest)
		} else {
			p.processTopLevelComment(comment)
		}
	}

	// At EOF with an open section.
	if p.currentSection != nil {
		p.addError(UnclosedSectionError{
			Start: *p.currentSection,
		})
	}
}

// processSuffixes parses a block that consists of domain suffixes and
// a metadata header.
func (p *parser) processSuffixes(block, header, rest Source) {
	s := Suffixes{
		Source: block,
	}

	var metadataSrc []string
	for _, line := range header.lineSources() {
		// TODO: s.Header should be a single Source for the entire
		// comment.
		s.Header = append(s.Header, line)
		if strings.HasPrefix(line.Text(), sectionMarkerPrefix) {
			p.addError(SectionInSuffixBlock{line})
		} else {
			// Trim the comment prefix in two steps, because some PSL
			// comments don't have whitepace between the // and the
			// following text.
			metadataSrc = append(metadataSrc, strings.TrimSpace(strings.TrimPrefix(line.Text(), "//")))
		}
	}

	// rest consists of suffixes and possibly inline comments.
	commentLine := func(line Source) bool { return strings.HasPrefix(line.Text(), "//") }
	rest.forEachRun(commentLine, func(block Source, isComment bool) {
		if isComment {
			for _, line := range block.lineSources() {
				if strings.HasPrefix(line.Text(), sectionMarkerPrefix) {
					p.addError(SectionInSuffixBlock{line})
				}
			}
			s.InlineComments = append(s.InlineComments, block)
		} else {
			// TODO: parse entries properly, for how we just
			// accumulate them as individual Sources, one per suffix.
			for _, entry := range block.lineSources() {
				s.Entries = append(s.Entries, entry)
			}
		}
	})

	enrichSuffixes(&s, metadataSrc)
	p.addBlock(s)
}

const sectionMarkerPrefix = "// ==="

// processTopLevelComment parses a block that has only comment lines,
// no suffixes. Some of those comments may be markers for the
// start/end of file sections.
func (p *parser) processTopLevelComment(block Source) {
	sectionLine := func(line Source) bool {
		return strings.HasPrefix(line.Text(), sectionMarkerPrefix)
	}
	block.forEachRun(sectionLine, func(block Source, isSectionLine bool) {
		if isSectionLine {
			for _, line := range block.lineSources() {
				p.processSectionMarker(line)
			}
		} else {
			p.addBlock(Comment{block})
		}
	})
}

// processSectionMarker parses line as a file section marker, and
// enforces correct start/end pairing.
func (p *parser) processSectionMarker(line Source) {
	// Trim here rather than in the caller, so that we still have the
	// complete input line available to use in errors.
	marker := strings.TrimPrefix(line.Text(), sectionMarkerPrefix)

	// Note hasTrailer gets used below to report an error if the
	// trailing "===" is missing. We delay reporting the error so that
	// if the entire line is invalid, we don't report both a
	// whole-line error and also an unterminated marker error.
	marker, hasTrailer := strings.CutSuffix(marker, "===")

	markerType, name, ok := strings.Cut(marker, " ")
	if !ok {
		// There are no spaces, markerType is the whole text between
		// the ===. Clear it out, so that the switch below goes to the
		// error case, otherwise "===BEGIN===" would be accepted as a
		// no-name section start.
		markerType = ""
	}

	// No matter what, we're going to output something that needs to
	// reference this line.
	src := line

	switch markerType {
	case "BEGIN":
		start := StartSection{
			Source: src,
			Name:   name,
		}
		if p.currentSection != nil {
			// Nested sections aren't allowed. Note the error and
			// continue parsing as if the previous section was closed
			// correctly before this one started.
			p.addError(NestedSectionError{
				Outer: *p.currentSection,
				Inner: start,
			})
		}
		if !hasTrailer {
			p.addError(UnterminatedSectionMarker{src})
		}
		p.currentSection = &start
		p.addBlock(start)
	case "END":
		end := EndSection{
			Source: src,
			Name:   name,
		}
		if p.currentSection == nil {
			// Rogue end marker. Note and continue parsing as if this
			// section name was correctly opened earlier.
			p.addError(UnstartedSectionError{
				End: end,
			})
		} else if p.currentSection.Name != name {
			// Mismatched start/end.
			p.addError(MismatchedSectionError{
				Start: *p.currentSection,
				End:   end,
			})
		}
		if !hasTrailer {
			p.addError(UnterminatedSectionMarker{src})
		}
		p.currentSection = nil
		p.addBlock(end)
	default:
		// Unknown kind of marker
		//
		// We want all non-whitespace bytes to be present in the
		// parsed output somewhere, so record this malformed line as a
		// Comment. Top-level comments are just freeform text, which
		// is technically correct here since this isn't a valid
		// section marker.
		p.addError(UnknownSectionMarker{src})
		p.addBlock(Comment{src})
	}
}

// addBlock adds b to p.File.Blocks.
func (p *parser) addBlock(b Block) {
	p.File.Blocks = append(p.File.Blocks, b)
}

// addError records err as a parse/validation error.
//
// If err matches a legacy exemption from current validation rules,
// err is recorded as a non-fatal warning instead.
func (p *parser) addError(err error) {
	if p.downgradeToWarning(err) {
		p.File.Warnings = append(p.File.Warnings, err)
	} else {
		p.File.Errors = append(p.File.Errors, err)
	}
}
