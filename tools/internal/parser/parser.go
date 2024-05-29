// Package parser implements a validating parser for the PSL files.
package parser

import (
	"net/mail"
	"net/url"
	"strings"
)

// Parse parses src as a PSL file and returns the parse result.
//
// The parser tries to keep going when it encounters errors. Parse and
// validation errors are accumulated in the Errors field of the
// returned File. A File with a non-empty Errors field is not a valid
// PSL file and may contain malformed data.
func Parse(src string) *File {
	return parseWithExceptions(src, downgradeToWarning)
}

func parseWithExceptions(src string, downgradeToWarning func(error) bool) *File {
	p := parser{
		downgradeToWarning: downgradeToWarning,
	}
	p.Parse(src)
	p.Validate()
	return &p.File
}

// parser is the state for a single PSL file parse.
type parser struct {
	// blockStart, if non-zero, is the line on which the current block began.
	blockStart int
	// blockEnd, if non-zero, is the line on which the last complete
	// block ended.
	blockEnd int
	// lines is the lines of source text between blockStart and
	// blockEnd.
	lines []string

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
func (p *parser) Parse(src string) {
	lines := strings.Split(src, "\n")
	// Add a final empty line to process, so that the block
	// consumption logic works even if there is no final empty line in
	// the source. This avoids the need for some final off-by-one
	// cleanup after the main parsing loop.
	lines = append(lines, "\n")

	// The top-level structure of a PSL file is blocks of non-empty
	// lines separated by one or more empty lines. This loop
	// accumulates one block at a time then gets consumeBlock() to
	// turn it into a parse output.
	for i, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			if len(p.lines) > 0 {
				p.blockEnd = i
				p.consumeBlock()
			}
			continue
		}
		if p.blockStart == 0 {
			p.blockStart = i + 1 // we 1-index, range 0-indexes
		}
		p.lines = append(p.lines, line)
	}

	// At EOF with an open section.
	if p.currentSection != nil {
		p.addError(UnclosedSectionError{
			Start: *p.currentSection,
		})
	}
}

// consumeBlock consumes the currently accumulated p.lines and
// produces one or more Blocks into p.File.Blocks.
//
// consumeBlock assumes that p.lines contains at least one line, and
// that p.blockStart and p.blockEnd are both non-zero. It resets all
// those fields to their zero value when it returns.
func (p *parser) consumeBlock() {
	defer func() {
		p.lines = nil
		p.blockStart = 0
		p.blockEnd = 0
	}()

	// Comment and Suffix blocks are distinguished by whether or not
	// there are any domain suffixes.
	var header, entries, comments []Source
	for i, l := range p.lines {
		src := Source{p.blockStart + i, p.blockStart + i, l}
		if !strings.HasPrefix(l, "//") {
			entries = append(entries, src)
		} else if len(entries) > 0 {
			comments = append(comments, src)
		} else {
			header = append(header, src)
		}
	}

	switch {
	case len(entries) > 0:
		s := Suffixes{
			Source:         p.blockSource(),
			Header:         header,
			Entries:        entries,
			InlineComments: comments,
		}
		p.enrichSuffixes(&s)
		p.addBlock(s)
	case strings.HasPrefix(header[0].Raw, sectionMarker):
		p.consumeSections()
	default:
		p.addBlock(Comment{
			Source: p.blockSource(),
		})
	}
}

// sectionMarker is the prefix of a comment line that indicates the
// start or end of a logical section of the file.
const beginMarker = "// ===BEGIN "
const endMarker = "// ===END "
const sectionMarker = "// ==="

// consumeComment generates a Comment block from the given lines. If
// any lines look like a section marker, those are passed to
// consumeSections.
func (p *parser) consumeComment() {
	for i, line := range p.lines {
		if !strings.HasPrefix(line, sectionMarker) {
			continue
		} else if strings.HasPrefix(line, beginMarker) || strings.HasPrefix(line, endMarker) {
			if i > 0 {
				p.addBlock(Comment{
					Source: Source{p.blockStart, p.blockStart + i - 1, strings.Join(p.lines[:i], "\n")},
				})
				p.lines = p.lines[i:]
				p.blockStart += i
			}
			p.consumeSections()
		} else {
			p.addError(UnknownSectionMarker{
				Line: Source{p.blockStart + i, p.blockStart + i, line},
			})
		}
	}

	if len(p.lines) > 0 {
		p.addBlock(Comment{
			Source: p.blockSource(),
		})
	}
}

// consumeSections looks for logical section start/end markers in
// p.lines and generates appropriate StartSection/EndSection blocks.
//
// If consumeSections encounters a non-marker line (a normal comment
// or a domain suffix), it stops and delegates further processing to
// consumeBlock, with the parser's state suitably adjusted to remove
// processed markers.
func (p *parser) consumeSections() {
	// A single comment block may interleave section markers and
	// freeform comments, and we want to translate that structure
	// faithfully. Consume consecutive marker lines, but stop as soon
	// as a non-marker line shows up.
	for len(p.lines) > 0 {
		marker := p.lines[0]
		src := Source{p.blockStart, p.blockStart, marker}

		marker = strings.TrimSuffix(marker, "===")
		if begin := strings.TrimPrefix(marker, beginMarker); begin != marker {
			start := StartSection{
				Source: src,
				Name:   begin,
			}
			if p.currentSection != nil {
				// Nested sections aren't allowed, note the error and
				// continue parsing as if the prior section had been
				// correctly closed.
				p.addError(NestedSectionError{
					Outer: *p.currentSection,
					Inner: start,
				})
			}
			p.currentSection = &start
			p.addBlock(start)
		} else if end := strings.TrimPrefix(marker, endMarker); end != marker {
			endSection := EndSection{
				Source: src,
				Name:   end,
			}
			if p.currentSection == nil {
				// Rogue end marker, note the error and continue
				// parsing as if the section had been correctly opened
				// earlier.
				p.addError(UnstartedSectionError{
					End: endSection,
				})
			} else if p.currentSection.Name != end {
				// Mismatched start/end. Note the error but keep going
				// as if the pairing was correct.
				p.addError(MismatchedSectionError{
					Start: *p.currentSection,
					End:   endSection,
				})
			}
			p.currentSection = nil
			p.addBlock(endSection)
		} else {
			// Looks like a section marker, but not a known form. Stop
			// processing section markers and punt back to comment
			// processing.
			//
			// consumeComment and consumeSections may recurse into
			// each other for particularly gnarly comments, but
			// outside of a deliberately malicious input the stack
			// depth remains acceptable - and malicious input just
			// causes a panic, not a safety issue.
			p.consumeComment()
			return
		}
		p.lines = p.lines[1:]
		p.blockStart++
	}
}

// enrichSuffixes extracts structured metadata from suffixes.Header
// and populates the appropriate fields of suffixes.
func (p *parser) enrichSuffixes(suffixes *Suffixes) {
	if len(suffixes.Header) == 0 {
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
	for _, line := range suffixes.Header {
		name, url, contact := splitNameish(trimComment(line.Raw))
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
		first := trimComment(suffixes.Header[0].Raw)
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
		for _, line := range suffixes.Header {
			if submitter := getSubmitter(trimComment(line.Raw)); submitter != nil {
				suffixes.Submitter = submitter
				break
			}
		}
	}
	if suffixes.Submitter == nil {
		for _, line := range suffixes.Header {
			if submitter, err := mail.ParseAddress(trimComment(line.Raw)); err == nil {
				suffixes.Submitter = submitter
				break
			}
		}
	}

	// Try to find a URL, if the previous step didn't find one. The
	// only remaining format we understand is a line with a URL by
	// itself.
	if suffixes.URL == nil {
		for _, line := range suffixes.Header {
			if u := getURL(trimComment(line.Raw)); u != nil {
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

	// A single entry uses the unicode fullwidth colon codepoint
	// (U+FF1A) instead of an ascii colon. Correct that before
	// attempting a parse.
	//
	// TODO: fix the source and delete this hack.
	if strings.Contains(line, "Future Versatile Group") {
		line = strings.Replace(line, "\uff1a", ":", -1)
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

	// Two PSL entries omit the scheme at the front of the URL, which
	// makes them invalid by getURL's standards.
	//
	// TODO: fix the source and delete this hack.
	if urlStr == "www.task.gda.pl/uslugi/dns" || urlStr == "hostyhosting.com" {
		urlStr = "https://" + urlStr
	}

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

	// One current entry is missing the closing chevron on the email,
	// which makes it an invalid address.
	//
	// TODO: fix the source and delete this hack.
	if strings.HasSuffix(line, "torproject.org") {
		if addr, err := mail.ParseAddress(line + ">"); err == nil {
			return addr
		}
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

// trimComment removes the leading // and outer whitespace from line.
func trimComment(line string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, "//"))
}

// blockSource returns a Source for p.lines.
func (p *parser) blockSource() Source {
	return Source{
		StartLine: p.blockStart,
		EndLine:   p.blockEnd,
		Raw:       strings.Join(p.lines, "\n"),
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
