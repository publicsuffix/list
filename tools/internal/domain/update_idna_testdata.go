//go:build ignore

// This script is run by `go generate` (see domains_test.go) to
// download a new copy of the IDNA test inputs. They are stored
// verbatim as provided by the Unicode Consortium to make it easy to
// verify that it's an unaltered file, and gets parsed for the
// information relevant to this package in domains_test.go.
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/natefinch/atomic"
	"golang.org/x/net/idna"
)

const (
	idnaTestVectorsURLPattern = "https://www.unicode.org/Public/idna/%s/IdnaTestV2.txt"
	idnaTestVectorsPath       = "testdata/idna_test_vectors.txt"
)

func main() {
	// New releases of Unicode can alter the outcome of existing
	// tests, so it's very important to use the test vectors for the
	// specific version of Unicode that x/net/idna uses.
	url := fmt.Sprintf(idnaTestVectorsURLPattern, idna.UnicodeVersion)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	} else if resp.StatusCode != http.StatusOK {
		log.Fatalf("Fetching %q: %v", url, err)
	}
	defer resp.Body.Close()

	if err := atomic.WriteFile(idnaTestVectorsPath, resp.Body); err != nil {
		log.Fatalf("Writing %q: %v", idnaTestVectorsPath, err)
	}
}
