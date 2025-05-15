package parser

import (
	"net/mail"
	"net/url"
	"strings"
)

// extractMaintainerInfo extracts structured maintainer metadata from
// comment.
func extractMaintainerInfo(comment *Comment) MaintainerInfo {
	if comment == nil || len(comment.Text) == 0 {
		return MaintainerInfo{MachineEditable: true}
	}

	var (
		ret = MaintainerInfo{
			MachineEditable: true,
		}
		lines             = comment.Text
		firstUnusableLine = -1
	)

	// The first line of metadata usually follows a standard
	// form. Handle that first, then scan through the rest of the
	// comment to find any further stuff.
	name, siteURL, email, ok := splitNameish(lines[0])
	if ok {
		ret.Name = name
		if siteURL != nil {
			ret.URLs = append(ret.URLs, siteURL)
		}
		if email != nil {
			ret.Maintainers = append(ret.Maintainers, email)
		}
		lines = lines[1:]
	}

	// Aside from the special first line, remaining lines could be
	// maintainer emails in a few formats, or URLs, or something
	// else. We accumulate everything we can parse, but also keep
	// track of whether the information is laid out such that we could
	// write the information back out without data loss (although not
	// necessarily in the exact same format).
	for i, line := range lines {
		lineUsed := false
		if emails := getSubmitters(line); len(emails) > 0 {
			ret.Maintainers = append(ret.Maintainers, emails...)
			lineUsed = true
		} else if email, err := mail.ParseAddress(line); err == nil {
			ret.Maintainers = append(ret.Maintainers, email)
			lineUsed = true
		} else if u := getURL(line); u != nil {
			ret.URLs = append(ret.URLs, u)
			lineUsed = true
		} else if i == 0 && ret.Name == "" {
			ret.Name = line
			lineUsed = true
		} else {
			ret.Other = append(ret.Other, line)
			if firstUnusableLine < 0 {
				firstUnusableLine = i + 1
			}
		}

		if lineUsed && firstUnusableLine >= 0 {
			// Parseable lines after non-parseable lines, we cannot
			// confidently write the data back out without dataloss.
			ret.MachineEditable = false
		}
	}

	return ret
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
//   - Any amount of whitespace on either side of the colon (or
//     fullwidth colon).
func splitNameish(line string) (name string, url *url.URL, submitter *mail.Address, ok bool) {
	if strings.HasPrefix(strings.ToLower(line), submittedBy) {
		// submitted-by lines are handled separately elsewhere, and
		// can be misinterpreted as entity names.
		return "", nil, nil, false
	}

	// Some older entries are of the form "entity name (url)".
	if strings.HasSuffix(line, ")") {
		if name, url, ok := splitNameAndURLInParens(line); ok {
			return name, url, nil, true
		}
	}

	name, rest, ok := strings.Cut(line, ":")
	if !ok {
		return "", nil, nil, false
	}

	// Clean up whitespace either side of the colon.
	name = strings.TrimSpace(name)
	rest = strings.TrimSpace(rest)

	if u := getURL(rest); u != nil {
		return name, u, nil, true
	} else if emails := getSubmitters(rest); len(emails) == 1 {
		return name, nil, emails[0], true
	}
	return "", nil, nil, false
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
func getSubmitters(line string) []*mail.Address {
	if strings.HasPrefix(strings.ToLower(line), submittedBy) {
		line = line[len(submittedBy):]
	}
	// Some entries read "Submitted by: ..." with an extra colon.
	line = strings.TrimLeft(line, ":")
	line = strings.TrimSpace(line)
	// Some ICANN domains lead with "Submitted by registry".
	line = strings.TrimPrefix(line, "registry ")

	var ret []*mail.Address
	emailStrs := strings.Split(line, " and ")

	fullyParsed := true
	for _, emailStr := range emailStrs {
		addr, err := mail.ParseAddress(emailStr)
		if err != nil {
			fullyParsed = false
			continue
		}
		ret = append(ret, addr)
	}

	if fullyParsed {
		// Found a way to consume the entire input, we're done.
		return ret
	}

	// One current entry uses old school email obfuscation to foil
	// spam bots, which makes it an invalid address.
	//
	// TODO: fix the source and delete this hack.
	if strings.Contains(line, "lohmus dot me") {
		cleaned := strings.Replace(line, " at ", "@", 1)
		cleaned = strings.Replace(cleaned, " dot ", ".", 1)
		if addr, err := mail.ParseAddress(cleaned); err == nil {
			return []*mail.Address{addr}
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
			return []*mail.Address{addr}
		}
	}

	return nil
}
