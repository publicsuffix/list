// Package parser implements a validating parser for the PSL files.
package parser

import (
	"net/mail"
	"net/url"
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

	p.enrichSuffixes(&s, metadataSrc)
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

// enrichSuffixes extracts structured metadata from suffixes.Header
// and populates the appropriate fields of suffixes.
func (p *parser) enrichSuffixes(suffixes *Suffixes, metadata []string) {
	if len(metadata) == 0 {
		return
	}

	// Try to find an entity name in the header. There are a few
	// possible ways this can appear, but the canonical is a first
	// header line of the form "<name>: <url>".
	//
	// If the canonical form is missing, a number of other variations
	// are tried in order to maximize the information we can extract
	// from the real PSL. Non-canonical representations may produce
	// validation errors in future, but currently do not.
	//
	// See splitNameish for a list of accepted alternate forms.
	for _, line := range metadata {
		name, url, contact := splitNameish(line)
		if name == "" {
			continue
		}

		suffixes.Entity = name
		if url != nil {
			suffixes.URL = url
		}
		if contact != nil {
			suffixes.Submitter = contact
		}
		break
	}
	if suffixes.Entity == "" {
		// Assume the first line is the entity name, if it's not
		// obviously something else.
		first := metadata[0]
		// "see also" is the first line of a number of ICANN TLD
		// sections.
		if getSubmitter(first) == nil && getURL(first) == nil && first != "see also" {
			suffixes.Entity = first
		}
	}

	// Try to find contact info, if the previous step didn't find
	// any. The only remaining formats we understand is a line with
	// "Submitted by <contact>", or failing that a parseable RFC5322
	// email on a line by itself.
	if suffixes.Submitter == nil {
		for _, line := range metadata {
			if submitter := getSubmitter(line); submitter != nil {
				suffixes.Submitter = submitter
				break
			}
		}
	}
	if suffixes.Submitter == nil {
		for _, line := range metadata {
			if submitter, err := mail.ParseAddress(line); err == nil {
				suffixes.Submitter = submitter
				break
			}
		}
	}

	// Try to find a URL, if the previous step didn't find one. The
	// only remaining format we understand is a line with a URL by
	// itself.
	if suffixes.URL == nil {
		for _, line := range metadata {
			if u := getURL(line); u != nil {
				suffixes.URL = u
				break
			}
		}
	}
}

// submittedBy is the conventional text that precedes email contact
// information in a PSL file. Most PSL entries say "Submitted by", but
// there are 4 entries that are lowercase, and so we do a
// case-insensitive comparison when looking for this marker.
const submittedBy = "submitted by"

// splitNameish tries to parse line in the form:
//
//	"<entity name>: <url or submitter email>"
//
// It returns the information it was able to extract. Returns all zero
// values if line does not conform to the expected form.
//
// As of 2024-06, a few legacy representations are also handled to
// improve compatibility with the existing PSL data:
//
//   - "<entity name> (<url>)", where the URL is sometimes allowed to
//     omit https://.
//   - "<entity name>: Submitted by <email address>", where the second
//     part is any variant accepted by getSubmitter.
//   - The canonical form, but with a unicode fullwidth colon (U+FF1A)
//     instead of a regular colon.
//   - Any amount of whitespace on either side of the colon (or
//     fullwidth colon).
func splitNameish(line string) (name string, url *url.URL, submitter *mail.Address) {
	if strings.HasPrefix(strings.ToLower(line), submittedBy) {
		// submitted-by lines are handled separately elsewhere, and
		// can be misinterpreted as entity names.
		return "", nil, nil
	}

	// Some older entries are of the form "entity name (url)".
	if strings.HasSuffix(line, ")") {
		if name, url, ok := splitNameAndURLInParens(line); ok {
			return name, url, nil
		}
	}

	name, rest, ok := strings.Cut(line, ":")
	if !ok {
		return "", nil, nil
	}

	// Clean up whitespace either side of the colon.
	name = strings.TrimSpace(name)
	rest = strings.TrimSpace(rest)

	if u := getURL(rest); u != nil {
		return name, u, nil
	} else if contact := getSubmitter(rest); contact != nil {
		return name, nil, contact
	}
	return "", nil, nil
}

// splitNameAndURLInParens tries to parse line in the form:
//
//	"<entity name> (<url>)"
//
// It returns the information it was able to extract, or ok=false if
// the line is not in the expected form.
func splitNameAndURLInParens(line string) (name string, url *url.URL, ok bool) {
	idx := strings.LastIndexByte(line, '(')
	if idx == -1 {
		return "", nil, false
	}
	name = strings.TrimSpace(line[:idx])
	urlStr := strings.TrimSpace(line[idx+1 : len(line)-1])

	if u := getURL(urlStr); u != nil {
		return name, u, true
	}

	return "", nil, false
}

// getURL tries to parse line as an HTTP/HTTPS URL.
// Returns the URL if line is a well formed URL and nothing but a URL,
// or nil otherwise.
func getURL(line string) *url.URL {
	// One PSL entry says "see <url>" instead of just a URL.
	//
	// TODO: fix the source and delete this hack.
	if strings.HasPrefix(line, "see https://www.information.aero") {
		line = strings.TrimPrefix(line, "see ")
	}

	u, err := url.Parse(line)
	if err != nil {
		return nil
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		// Caller might have split https://foo.com into [https :
		// //foo.com], and the last part is a valid scheme-relative
		// URL. Only accept parses that feature an explicit http(s)
		// scheme.
		return nil
	}

	return u
}

// getSubmitter tries to parse line as a submitter email line, usually:
//
//	Submitted by Person Name <person.email@example.com>
//
// To improve compatibility, a few legacy freeform styles are also
// attempted if the one above fails.
//
// Returns the parsed RFC 5322 address, or nil if line does not
// conform to the expected shape.
func getSubmitter(line string) *mail.Address {
	if !strings.HasPrefix(strings.ToLower(line), submittedBy) {
		return nil
	}
	line = line[len(submittedBy):]
	// Some entries read "Submitted by: ..." with an extra colon.
	line = strings.TrimLeft(line, ":")
	line = strings.TrimSpace(line)
	// Some ICANN domains lead with "Submitted by registry".
	line = strings.TrimLeft(line, "registry ")

	if addr, err := mail.ParseAddress(line); err == nil {
		return addr
	}

	// One current entry uses old school email obfuscation to foil
	// spam bots, which makes it an invalid address.
	//
	// TODO: fix the source and delete this hack.
	if strings.Contains(line, "lohmus dot me") {
		cleaned := strings.Replace(line, " at ", "@", 1)
		cleaned = strings.Replace(cleaned, " dot ", ".", 1)
		if addr, err := mail.ParseAddress(cleaned); err == nil {
			return addr
		}
	}

	// The normal form failed but there is a "submitted by". If the
	// last word is an email address, assume the remainder is a name.
	fs := strings.Fields(line)
	if len(fs) > 0 {
		if addr, err := mail.ParseAddress(fs[len(fs)-1]); err == nil {
			name := strings.Join(fs[:len(fs)-1], " ")
			name = strings.Trim(name, " ,:")
			addr.Name = name
			return addr
		}
	}

	return nil
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
