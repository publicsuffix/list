package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/creachadair/taskgroup"
	"github.com/publicsuffix/list/tools/internal/domain"
)

// validateSecurityTxt checks that every changed Suffix and Wildcard
// under b publishes a valid RFC 9116 security.txt file. Wildcards are
// checked at their base domain.
func validateSecurityTxt(ctx context.Context, b Block, client *http.Client, now time.Time) []error {
	// concurrentHTTPRequests is the maximum number of in-flight
	// security.txt fetches. Each fetch goes to a different third-party
	// server, so this mostly bounds local resource usage.
	const concurrentHTTPRequests = 10

	var blocks []Block
	for _, suf := range BlocksOfType[*Suffix](b) {
		if suf.Changed() {
			blocks = append(blocks, suf)
		}
	}
	for _, wild := range BlocksOfType[*Wildcard](b) {
		if wild.Changed() {
			blocks = append(blocks, wild)
		}
	}

	// Each task writes only its own slot, so results are reported in
	// the same deterministic order as blocks.
	results := make([]error, len(blocks))
	group, start := taskgroup.New(nil).Limit(concurrentHTTPRequests)
	for i, block := range blocks {
		start(func() error {
			dom := suffixDomain(block)
			log.Printf("Checking %s", securityTxtURL(dom))
			text, err := fetchSecurityTxt(ctx, client, dom)
			if err != nil {
				results[i] = ErrMissingSecurityTxt{block, err}
			} else if err := checkSecurityTxt(text, now); err != nil {
				results[i] = ErrInvalidSecurityTxt{block, err}
			}
			return nil
		})
	}
	group.Wait()

	var errs []error
	for _, err := range results {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// securityTxtURL returns the URL at which dom's security.txt file is
// expected to be published.
func securityTxtURL(dom domain.Name) string {
	return "https://" + dom.ASCIIString() + "/.well-known/security.txt"
}

// fetchSecurityTxt returns the body of dom's security.txt file, or an
// error if dom does not serve one. RFC 9116 requires the file to be
// served as text/plain, so other content types, such as HTML pages
// returned in place of a 404, are treated as no file.
func fetchSecurityTxt(ctx context.Context, client *http.Client, dom domain.Name) (string, error) {
	// maxSecurityTxtSize bounds how much of a response body is read,
	// so that a misbehaving server cannot exhaust memory.
	const maxSecurityTxtSize = 64 << 10

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, securityTxtURL(dom), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if mediaType, _, _ := mime.ParseMediaType(contentType); mediaType != "text/plain" {
		return "", fmt.Errorf("served as %q, not text/plain", contentType)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSecurityTxtSize))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// checkSecurityTxt verifies that text is a security.txt file with at
// least one Contact field and an Expires field that is not in the
// past relative to now, as required by RFC 9116.
func checkSecurityTxt(text string, now time.Time) error {
	var hasContact bool
	var expires string

	for line := range strings.Lines(text) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		// RFC 9116 field names are case-insensitive.
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "contact":
			hasContact = true
		case "expires":
			expires = strings.TrimSpace(value)
		}
	}

	if !hasContact {
		return errors.New("no Contact field")
	}
	if expires == "" {
		return errors.New("no Expires field")
	}
	expiry, err := time.Parse(time.RFC3339, expires)
	if err != nil {
		return fmt.Errorf("invalid Expires value %q", expires)
	}
	if expiry.Before(now) {
		return fmt.Errorf("expired on %s", expires)
	}
	return nil
}
