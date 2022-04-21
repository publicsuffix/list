// newgtlds is a utility command that downloads the list of gTLDs from ICANN
// and formats it into the PSL format, writing to stdout.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

const (
	// ICANN_GTLD_JSON_URL is the URL for the ICANN gTLD JSON registry (version
	// 2). See https://www.icann.org/resources/pages/registries/registries-en for
	// more information.
	ICANN_GTLD_JSON_URL = "https://www.icann.org/resources/registries/gtlds/v2/gtlds.json"
	// IANA_TLDS_TXT_URL is the URL for the IANA "Public Suffix List" of TLDs
	// in the ICP-3 Root - including new ccTLDs, EBRERO gTLDS or things not in
	// the JSON File above that should be included in the PSL.  Note: UPPERCASE
	IANA_TLDS_TXT_URL = "http://data.iana.org/TLD/tlds-alpha-by-domain.txt"
	// PSL_GTLDS_SECTION_HEADER marks the start of the newGTLDs section of the
	// overall public suffix dat file.
	PSL_GTLDS_SECTION_HEADER = "// newGTLDs"
	// PSL_GTLDS_SECTION_FOOTER marks the end of the newGTLDs section of the
	// overall public suffix dat file.
	PSL_GTLDS_SECTION_FOOTER = "// ===END ICANN DOMAINS==="
)

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

	// pslHeaderTemplate is a parsed text/template instance for rendering the header
	// before the data rendered with the pslTemplate. We use two separate templates
	// so that we can avoid having a variable date stamp in the pslTemplate, allowing
	// us to easily check that the data in the current .dat file is unchanged from
	// what we render when there are no updates to add.
	//
	// Expected template data:
	//   URL - the string URL that the data was fetched from.
	//   Date - the time.Date that the data was fetched.
	//   DateFormat - the format string to use with the date.
	pslHeaderTemplate = template.Must(template.New("public-suffix-list-gtlds-header").Parse(`
// List of new gTLDs imported from {{ .URL }} on {{ .Date.Format .DateFormat }}
// This list is auto-generated, don't edit it manually.`))

	// pslTemplate is a parsed text/template instance for rendering a list of pslEntry
	// objects in the format used by the public suffix list.
	//
	// It expects the following template data:
	//   Entries - a list of pslEntry objects.
	pslTemplate = template.Must(
		template.New("public-suffix-list-gtlds").Parse(`
{{- range .Entries }}
{{- .Comment }}
{{ printf "%s\n" .ULabel }}
{{ end }}`))
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
	// RemovalDate indicates the date the gTLD delegation was removed from the
	// root zones.
	RemovalDate string
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

// gTLDDatSpan represents the span between the PSL_GTLD_SECTION_HEADER and
// the PSL_GTLDS_SECTION_FOOTER in the PSL dat file.
type gTLDDatSpan struct {
	startIndex int
	endIndex   int
}

var (
	errNoHeader = fmt.Errorf("did not find expected header line %q",
		PSL_GTLDS_SECTION_HEADER)
	errMultipleHeaders = fmt.Errorf("found expected header line %q more than once",
		PSL_GTLDS_SECTION_HEADER)
	errNoFooter = fmt.Errorf("did not find expected footer line %q",
		PSL_GTLDS_SECTION_FOOTER)
)

type errInvertedSpan struct {
	span gTLDDatSpan
}

func (e errInvertedSpan) Error() string {
	return fmt.Sprintf(
		"found footer line %q before header line %q (index %d vs %d)",
		PSL_GTLDS_SECTION_FOOTER, PSL_GTLDS_SECTION_HEADER,
		e.span.endIndex, e.span.startIndex)
}

// validate checks that a given gTLDDatSpan is sensible. It returns an err if
// the span is nil, if the start or end index haven't been set to > 0, or if the
// end index is <= the the start index.
func (s gTLDDatSpan) validate() error {
	if s.startIndex <= 0 {
		return errNoHeader
	}
	if s.endIndex <= 0 {
		return errNoFooter
	}
	if s.endIndex <= s.startIndex {
		return errInvertedSpan{span: s}
	}
	return nil
}

// datFile holds the individual lines read from the public suffix list dat file and
// the span that holds the gTLD specific data section. It supports reading the
// gTLD specific data, and replacing it.
type datFile struct {
	// lines holds the datfile contents split by "\n"
	lines []string
	// gTLDSpan holds the indexes where the gTLD data can be found in lines.
	gTLDSpan gTLDDatSpan
}

type errSpanOutOfBounds struct {
	span     gTLDDatSpan
	numLines int
}

func (e errSpanOutOfBounds) Error() string {
	return fmt.Sprintf(
		"span out of bounds: start index %d, end index %d, number of lines %d",
		e.span.startIndex, e.span.endIndex, e.numLines)
}

// validate validates the state of the datFile. It returns an error if
// the gTLD span validate() returns an error, or if gTLD span endIndex is >= the
// number of lines in the file.
func (d datFile) validate() error {
	if err := d.gTLDSpan.validate(); err != nil {
		return err
	}
	if d.gTLDSpan.endIndex >= len(d.lines) {
		return errSpanOutOfBounds{span: d.gTLDSpan, numLines: len(d.lines)}
	}
	return nil
}

// getGTLDLines returns the lines from the dat file within the gTLD data span,
// or an error if the span isn't valid for the dat file.
func (d datFile) getGTLDLines() ([]string, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	return d.lines[d.gTLDSpan.startIndex:d.gTLDSpan.endIndex], nil
}

// ReplaceGTLDContent updates the dat file's lines to replace the gTLD data span
// with new content.
func (d *datFile) ReplaceGTLDContent(content string) error {
	if err := d.validate(); err != nil {
		return err
	}

	contentLines := strings.Split(content, "\n")
	beforeLines := d.lines[0:d.gTLDSpan.startIndex]
	afterLines := d.lines[d.gTLDSpan.endIndex:]
	newLines := append(beforeLines, append(contentLines, afterLines...)...)

	// Update the span based on the new content length
	d.gTLDSpan.endIndex = len(beforeLines) + len(contentLines)
	// and update the data file lines
	d.lines = newLines
	return nil
}

// String returns the dat file's lines joined together.
func (d datFile) String() string {
	return strings.Join(d.lines, "\n")
}

// readDatFile reads the contents of the PSL dat file from the provided path
// and returns a representation holding all of the lines and the span where the gTLD
// data is found within the dat file. An error is returned if the file can't be read
// or if the gTLD data span can't be found or is invalid.
func readDatFile(datFilePath string) (*datFile, error) {
	pslDatBytes, err := ioutil.ReadFile(datFilePath)
	if err != nil {
		return nil, err
	}
	return readDatFileContent(string(pslDatBytes))
}

func readDatFileContent(pslData string) (*datFile, error) {
	pslDatLines := strings.Split(pslData, "\n")

	headerIndex, footerIndex := 0, 0
	for i := 0; i < len(pslDatLines); i++ {
		line := pslDatLines[i]

		if line == PSL_GTLDS_SECTION_HEADER && headerIndex == 0 {
			// If the line matches the header and we haven't seen the header yet, capture
			// the index
			headerIndex = i
		} else if line == PSL_GTLDS_SECTION_HEADER && headerIndex != 0 {
			// If the line matches the header and we've already seen the header return
			// an error. This is unexpected.
			return nil, errMultipleHeaders
		} else if line == PSL_GTLDS_SECTION_FOOTER && footerIndex == 0 {
			// If the line matches the footer, capture the index. We don't need
			// to consider the case where we've already seen a footer because we break
			// below when we have both a header and footer index.
			footerIndex = i
		}

		// Break when we have found one header and one footer.
		if headerIndex != 0 && footerIndex != 0 {
			break
		}
	}

	if headerIndex == 0 {
		return nil, errNoHeader
	} else if footerIndex == 0 {
		return nil, errNoFooter
	}

	datFile := &datFile{
		lines: pslDatLines,
		gTLDSpan: gTLDDatSpan{
			startIndex: headerIndex + 1,
			endIndex:   footerIndex,
		},
	}
	if err := datFile.validate(); err != nil {
		return nil, err
	}

	return datFile, nil
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
// ContractTerminated equal to true, or a non-empty RemovalDate.
func filterGTLDs(entries []*pslEntry) []*pslEntry {
	var filtered []*pslEntry
	for _, entry := range entries {
		if _, isLegacy := legacyGTLDs[entry.ALabel]; isLegacy {
			continue
		}
		if entry.ContractTerminated {
			continue
		}
		if entry.RemovalDate != "" {
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

// renderTemplate renders the given template to the provided writer, using the
// templateData, or returns an error.
func renderTemplate(writer io.Writer, template *template.Template, templateData interface{}) error {
	var buf bytes.Buffer
	if err := template.Execute(&buf, templateData); err != nil {
		return err
	}

	_, err := writer.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// clock is a small interface that lets us mock time in unit tests.
type clock interface {
	Now() time.Time
}

// realClock is an implementation of clock that uses time.Now() natively.
type realClock struct{}

// Now returns the current time.Time using the system clock.
func (c realClock) Now() time.Time {
	return time.Now()
}

// renderHeader renders the pslHeaderTemplate to the writer or returns an error. The
// provided clock instance is used for the header last update timestamp. If no
// clk instance is provided realClock is used.
func renderHeader(writer io.Writer, clk clock) error {
	if clk == nil {
		clk = &realClock{}
	}
	templateData := struct {
		URL        string
		Date       time.Time
		DateFormat string
	}{
		URL:        ICANN_GTLD_JSON_URL,
		Date:       clk.Now().UTC(),
		DateFormat: time.RFC3339,
	}

	return renderTemplate(writer, pslHeaderTemplate, templateData)
}

// renderData renders the given list of pslEntry objects using the pslTemplate.
// The rendered template data is written to the provided writer or an error is
// returned.
func renderData(writer io.Writer, entries []*pslEntry) error {
	templateData := struct {
		Entries []*pslEntry
	}{
		Entries: entries,
	}

	return renderTemplate(writer, pslTemplate, templateData)
}

// Process handles updating a datFile with new gTLD content. If there are no
// gTLD updates the existing dat file's contents will be returned. If there are
// updates, the new updates will be spliced into place and the updated file contents
// returned.
func process(datFile *datFile, dataURL string, clk clock) (string, error) {
	// Get the lines for the gTLD data span - this includes both the header with the
	// date and the actual gTLD entries.
	spanLines, err := datFile.getGTLDLines()
	if err != nil {
		return "", err
	}

	// Render a new header for the gTLD data.
	var newHeaderBuf strings.Builder
	if err := renderHeader(&newHeaderBuf, clk); err != nil {
		return "", err
	}

	// Figure out how many lines the header with the dynamic date is.
	newHeaderLines := strings.Split(newHeaderBuf.String(), "\n")
	headerLen := len(newHeaderLines)

	// We should have at least that many lines in the existing span data.
	if len(spanLines) <= headerLen {
		return "", errors.New("gtld span data was too small, missing header?")
	}

	// The gTLD data can be found by skipping the header lines
	existingData := strings.Join(spanLines[headerLen:], "\n")

	// Fetch new PSL entries.
	entries, err := getPSLEntries(dataURL)
	if err != nil {
		return "", err
	}

	// Render the new gTLD PSL section with the new entries.
	var newDataBuf strings.Builder
	if err := renderData(&newDataBuf, entries); err != nil {
		return "", err
	}

	// If the newly rendered data doesn't match the existing data then we want to
	// update the dat file content by replacing the old span with the new content.
	if newDataBuf.String() != existingData {
		newContent := newHeaderBuf.String() + "\n" + newDataBuf.String()
		if err := datFile.ReplaceGTLDContent(newContent); err != nil {
			return "", err
		}
	}

	return datFile.String(), nil
}

func main() {
	ifErrQuit := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error updating gTLD data: %v\n", err)
			os.Exit(1)
		}
	}

	pslDatFile := flag.String(
		"psl-dat-file",
		"public_suffix_list.dat",
		"file path to the public_suffix_list.dat data file to be updated with new gTLDs")

	overwrite := flag.Bool(
		"overwrite",
		false,
		"overwrite -psl-dat-file with the new data instead of printing to stdout")

	// Parse CLI flags.
	flag.Parse()

	// Read the existing file content and find the span that contains the gTLD data.
	datFile, err := readDatFile(*pslDatFile)
	ifErrQuit(err)

	// Process the dat file.
	content, err := process(datFile, ICANN_GTLD_JSON_URL, nil)
	ifErrQuit(err)

	// If we're not overwriting the file, print the content to stdout.
	if !*overwrite {
		fmt.Println(content)
		os.Exit(0)
	}

	// Otherwise print nothing to stdout and write the content over the exiting
	// pslDatFile path we read earlier.
	err = ioutil.WriteFile(*pslDatFile, []byte(content), 0644)
	ifErrQuit(err)
}
