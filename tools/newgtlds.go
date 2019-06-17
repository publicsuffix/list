// newgtlds is a utility command that downloads the list of gTLDs from ICANN
// and formats it into the PSL format, writing to stdout.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

// ICANN_GTLD_JSON_URL is the URL for the ICANN gTLD JSON registry (version
// 2). See https://www.icann.org/resources/pages/registries/registries-en for
// more information.
const ICANN_GTLD_JSON_URL = "https://www.icann.org/resources/registries/gtlds/v2/gtlds.json"

var (
	// legacyGTLDs are gTLDs that predate ICANN's new gTLD program. These legacy
	// gTLDs are present in the ICANN_GTLD_JSON_URL data but we do not want to
	// include them in the new gTLD section of the PSL data because it will create
	// duplicates with existing entries alongside registry-reserved second level
	// domains present in the PSL data. Entries present in legacyGTLDs will not be
	// output by this tool when generating the new gTLD data.
	legacyGTLDs = map[string]bool{
		"aero":   true,
		"asia":   true,
		"biz":    true,
		"cat":    true,
		"com":    true,
		"coop":   true,
		"info":   true,
		"jobs":   true,
		"mobi":   true,
		"museum": true,
		"name":   true,
		"net":    true,
		"org":    true,
		"post":   true,
		"pro":    true,
		"tel":    true,
		"xxx":    true,
	}

	// pslTemplate is a parsed text/template instance for rendering a list of pslEntry
	// objects in the format used by the public suffix list.
	//
	// It expects the following template data:
	//   URL - the string URL that the data was fetched from.
	//   Date - the time.Date that the data was fetched.
	//   Entries - a list of pslEntry objects.
	pslTemplate = template.Must(template.New("public-suffix-list-gtlds").Parse(`
// List of new gTLDs imported from {{ .URL }} on {{ .Date.Format "2006-01-02T15:04:05Z07:00" }}
// This list is auto-generated, don't edit it manually.

{{- range .Entries }}
{{ .Comment }}
{{ printf "%s\n" .ULabel}}
{{- end }}
`))
)

// pslEntry is a struct matching a subset of the gTLD data fields present in
// each object entry of the "GLTDs" array from ICANN_GTLD_JSON_URL.
type pslEntry struct {
	// ALabel contains the ASCII gTLD name. For internationalized gTLDs the GTLD
	// field is expressed in punycode.
	ALabel string `json:"gTLD"`
	// ULabel contains the unicode representation of the gTLD name. When the gTLD
	// ULabel in the ICANN gTLD data is empty (e.g for an ASCII gTLD like
	// '.pizza') the PSL entry will use the ALabel as the ULabel.
	ULabel string
	// RegistryOperator holds the name of the registry operator that operates the
	// gTLD (may be empty).
	RegistryOperator string
	// DateOfContractSignature holds the date the gTLD contract was signed (may be empty).
	DateOfContractSignature string
	// ContractTerminated indicates whether the contract has been terminated by
	// ICANN. When rendered by the pslTemplate only entries with
	// ContractTerminated = false are included.
	ContractTerminated bool
}

// normalize will normalize a pslEntry by mutating it in place to trim the
// string fields of whitespace and by populating the ULabel with the ALabel if
// the ULabel is empty.
func (e *pslEntry) normalize() {
	e.ALabel = strings.TrimSpace(e.ALabel)
	e.ULabel = strings.TrimSpace(e.ULabel)
	e.RegistryOperator = strings.TrimSpace(e.RegistryOperator)
	e.DateOfContractSignature = strings.TrimSpace(e.DateOfContractSignature)

	// If there is no explicit uLabel use the gTLD as the uLabel.
	if e.ULabel == "" {
		e.ULabel = e.ALabel
	}
}

// Comment generates a comment string for the pslEntry. This string has a `//`
// prefix and matches one of the following two forms.
//
// If the registry operator field is empty the comment will be of the form:
//
//    '// <ALabel> : <DateOfContractSignature>'
//
// If the registry operator field is not empty the comment will be of the form:
//
//    '// <ALabel> : <DateOfContractSignature> <RegistryOperator>'
//
// In both cases the <DateOfContractSignature> may be empty.
func (e pslEntry) Comment() string {
	parts := []string{
		"//",
		e.ALabel,
		":",
		e.DateOfContractSignature,
	}
	// Avoid two trailing spaces if registry operator is empty
	if e.RegistryOperator != "" {
		parts = append(parts, e.RegistryOperator)
	}
	return strings.Join(parts, " ")
}

// getData performs a HTTP GET request to the given URL and returns the
// response body bytes or returns an error. An HTTP response code other than
// http.StatusOK (200) is considered to be an error.
func getData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code fetching data "+
			"from %q : expected status %d got %d",
			url, http.StatusOK, resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

// filterGTLDs removes entries that are present in the legacyGTLDs map or have
// ContractTerminated equal to true.
func filterGTLDs(entries []*pslEntry) []*pslEntry {
	var filtered []*pslEntry
	for _, entry := range entries {
		if _, isLegacy := legacyGTLDs[entry.ALabel]; isLegacy {
			continue
		}
		if entry.ContractTerminated {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

// getPSLEntries fetches a list of pslEntry objects (or returns an error) by:
//   1. getting the raw JSON data from the provided url string.
//   2. unmarshaling the JSON data to create pslEntry objects.
//   3. normalizing the pslEntry objects.
//   4. filtering out any legacy or contract terminated gTLDs
//
// If there are no pslEntry objects after unmarshaling the data in step 2 or
// filtering the gTLDs in step 4 it is considered an error condition.
func getPSLEntries(url string) ([]*pslEntry, error) {
	respBody, err := getData(url)
	if err != nil {
		return nil, err
	}

	var results struct {
		GTLDs []*pslEntry
	}
	if err := json.Unmarshal(respBody, &results); err != nil {
		return nil, fmt.Errorf(
			"unmarshaling ICANN gTLD JSON data: %v", err)
	}

	// We expect there to always be GTLD data. If there was none after unmarshaling
	// then its likely the data format has changed or something else has gone wrong.
	if len(results.GTLDs) == 0 {
		return nil, errors.New("found no gTLD information after unmarshaling")
	}

	// Normalize each tldEntry. This will remove leading/trailing whitespace and
	// populate the ULabel with the ALabel if the entry has no ULabel.
	for _, tldEntry := range results.GTLDs {
		tldEntry.normalize()
	}

	filtered := filterGTLDs(results.GTLDs)
	if len(filtered) == 0 {
		return nil, errors.New(
			"found no gTLD information after removing legacy and contract terminated gTLDs")
	}
	return filtered, nil
}

// renderData renders the given list of pslEntry objects using the pslTemplate.
// The rendered template data is written to the provided writer.
func renderData(entries []*pslEntry, writer io.Writer) error {
	templateData := struct {
		URL     string
		Date    time.Time
		Entries []*pslEntry
	}{
		URL:     ICANN_GTLD_JSON_URL,
		Date:    time.Now(),
		Entries: entries,
	}

	var buf bytes.Buffer
	if err := pslTemplate.Execute(&buf, templateData); err != nil {
		return err
	}

	_, err := writer.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// main will fetch the PSL entires from the ICANN gTLD JSON registry, parse
// them, normalize them, remove legacy and terminated gTLDs, and finally render
// them with the pslTemplate, printing the results to standard out.
func main() {
	ifErrQuit := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error updating gTLD data: %v\n", err)
			os.Exit(1)
		}
	}

	entries, err := getPSLEntries(ICANN_GTLD_JSON_URL)
	ifErrQuit(err)

	err = renderData(entries, os.Stdout)
	ifErrQuit(err)
}
