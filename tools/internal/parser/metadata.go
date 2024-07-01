package parser

import (
	"net/mail"
	"net/url"
	"strings"
)

// enrichSuffixes extracts structured metadata from metadata and
// populates the appropriate fields of suffixes.
func enrichSuffixes(suffixes *Suffixes, metadata []string) {
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
