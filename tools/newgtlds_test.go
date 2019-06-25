package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestEntryNormalize(t *testing.T) {
	testCases := []struct {
		name          string
		inputEntry    pslEntry
		expectedEntry pslEntry
	}{
		{
			name: "already normalized",
			inputEntry: pslEntry{
				ALabel:                  "cpu",
				ULabel:                  "ｃｐｕ",
				DateOfContractSignature: "2019-06-13",
				RegistryOperator:        "@cpu's bargain gTLD emporium",
			},
			expectedEntry: pslEntry{
				ALabel:                  "cpu",
				ULabel:                  "ｃｐｕ",
				DateOfContractSignature: "2019-06-13",
				RegistryOperator:        "@cpu's bargain gTLD emporium",
			},
		},
		{
			name: "extra whitespace",
			inputEntry: pslEntry{
				ALabel:                  "  cpu    ",
				ULabel:                  "   ｃｐｕ   ",
				DateOfContractSignature: "   2019-06-13    ",
				RegistryOperator: "     @cpu's bargain gTLD emporium " +
					"(now with bonus whitespace)    ",
			},
			expectedEntry: pslEntry{
				ALabel:                  "cpu",
				ULabel:                  "ｃｐｕ",
				DateOfContractSignature: "2019-06-13",
				RegistryOperator: "@cpu's bargain gTLD emporium " +
					"(now with bonus whitespace)",
			},
		},
		{
			name: "no explicit uLabel",
			inputEntry: pslEntry{
				ALabel:                  "cpu",
				DateOfContractSignature: "2019-06-13",
				RegistryOperator:        "@cpu's bargain gTLD emporium",
			},
			expectedEntry: pslEntry{
				ALabel:                  "cpu",
				ULabel:                  "cpu",
				DateOfContractSignature: "2019-06-13",
				RegistryOperator:        "@cpu's bargain gTLD emporium",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := &tc.inputEntry
			entry.normalize()
			if deepEqual := reflect.DeepEqual(*entry, tc.expectedEntry); !deepEqual {
				t.Errorf("entry did not match expected after normalization. %v vs %v",
					*entry, tc.expectedEntry)
			}
		})
	}
}

func TestEntryComment(t *testing.T) {
	testCases := []struct {
		name     string
		entry    pslEntry
		expected string
	}{
		{
			name: "Full entry",
			entry: pslEntry{
				ALabel:                  "cpu",
				DateOfContractSignature: "2019-06-13",
				RegistryOperator:        "@cpu's bargain gTLD emporium",
			},
			expected: "// cpu : 2019-06-13 @cpu's bargain gTLD emporium",
		},
		{
			name: "Entry with empty contract signature date and operator",
			entry: pslEntry{
				ALabel: "cpu",
			},
			expected: "// cpu : ",
		},
		{
			name: "Entry with empty contract signature and non-empty operator",
			entry: pslEntry{
				ALabel:           "cpu",
				RegistryOperator: "@cpu's bargain gTLD emporium",
			},
			expected: "// cpu :  @cpu's bargain gTLD emporium",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := tc.entry.Comment(); actual != tc.expected {
				t.Errorf("entry %v Comment() == %q expected == %q",
					tc.entry, actual, tc.expected)
			}
		})
	}
}

type badStatusHandler struct{}

func (h *badStatusHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusUnavailableForLegalReasons)
	w.Write([]byte("sorry"))
}

func TestGetData(t *testing.T) {
	handler := &badStatusHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	// NOTE: TestGetData only tests the handling of non-200 status codes in
	// getData as anything else is just testing stdlib code.
	resp, err := getData(server.URL)
	if err == nil {
		t.Error("expected getData() to a bad status handler server to return an " +
			"error, got nil")
	}
	if resp != nil {
		t.Errorf("expected getData() to a bad status handler server to return a "+
			"nil response body byte slice, got: %v",
			resp)
	}
}

type mockHandler struct {
	respData []byte
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Write(h.respData)
}

func TestGetPSLEntries(t *testing.T) {
	mockData := struct {
		GTLDs []pslEntry
	}{
		GTLDs: []pslEntry{
			{
				ALabel:                  "ceepeeyou",
				DateOfContractSignature: "2099-06-13",
				RegistryOperator:        "@cpu's bargain gTLD emporium",
			},
			{
				// NOTE: we include whitespace in this entry to test that normalization
				// occurs.
				ALabel:                  "  cpu    ",
				ULabel:                  "   ｃｐｕ   ",
				DateOfContractSignature: "   2019-06-13    ",
				RegistryOperator: "     @cpu's bargain gTLD emporium " +
					"(now with bonus whitespace)    ",
			},
			{
				// NOTE: we include a legacy gTLD here to test that filtering of legacy
				// gTLDs occurs.
				ALabel:                  "aero",
				DateOfContractSignature: "1999-10-31",
				RegistryOperator:        "Department of Historical Baggage and Technical Debt",
			},
			{
				ALabel:                  "terminated",
				DateOfContractSignature: "1987-10-31",
				// NOTE: we include a contract terminated = true entry here to test that
				// filtering of terminated entries occurs.
				ContractTerminated: true,
			},
		},
	}
	// NOTE: swallowing the possible err return here because the mock data is
	// assumed to be static/correct and it simplifies the handler.
	jsonBytes, _ := json.Marshal(mockData)

	expectedEntries := []pslEntry{
		{
			ALabel:                  "ceepeeyou",
			ULabel:                  "ceepeeyou",
			DateOfContractSignature: "2099-06-13",
			RegistryOperator:        "@cpu's bargain gTLD emporium",
		},
		{
			ALabel:                  "cpu",
			ULabel:                  "ｃｐｕ",
			DateOfContractSignature: "2019-06-13",
			RegistryOperator: "@cpu's bargain gTLD emporium " +
				"(now with bonus whitespace)",
		},
	}

	handler := &mockHandler{jsonBytes}
	server := httptest.NewServer(handler)
	defer server.Close()

	entries, err := getPSLEntries(server.URL)
	if err != nil {
		t.Fatalf("expected no error from getPSLEntries with mockHandler. Got %v",
			err)
	}

	if len(entries) != len(expectedEntries) {
		t.Fatalf("expected %d entries from getPSLEntries with mockHandler. Got %d",
			len(expectedEntries),
			len(entries))
	}

	for i, entry := range entries {
		if deepEqual := reflect.DeepEqual(*entry, expectedEntries[i]); !deepEqual {
			t.Errorf("getPSLEntries() entry index %d was %#v, expected %#v",
				i,
				*entry,
				expectedEntries[i])
		}
	}
}

func TestGetPSLEntriesEmptyResults(t *testing.T) {
	// Mock an empty result
	mockData := struct {
		GTLDs []pslEntry
	}{}

	// NOTE: swallowing the possible err return here because the mock data is
	// assumed to be static/correct and it simplifies the handler.
	jsonBytes, _ := json.Marshal(mockData)

	handler := &mockHandler{jsonBytes}
	server := httptest.NewServer(handler)
	defer server.Close()

	_, err := getPSLEntries(server.URL)
	if err == nil {
		t.Error("expected error from getPSLEntries with empty results mockHandler. Got nil")
	}
}

func TestGetPSLEntriesEmptyFilteredResults(t *testing.T) {
	// Mock data that will be filtered to an empty list
	mockData := struct {
		GTLDs []pslEntry
	}{
		GTLDs: []pslEntry{
			{
				// NOTE: GTLD matches a legacyGTLDs map entry to ensure filtering.
				ALabel:                  "aero",
				DateOfContractSignature: "1999-10-31",
				RegistryOperator:        "Department of Historical Baggage and Technical Debt",
			},
			{
				ALabel:                  "terminated",
				DateOfContractSignature: "1987-10-31",
				// NOTE: Setting ContractTerminated to ensure filtering.
				ContractTerminated: true,
			},
		},
	}

	// NOTE: swallowing the possible err return here because the mock data is
	// assumed to be static/correct and it simplifies the handler.
	jsonBytes, _ := json.Marshal(mockData)

	handler := &mockHandler{jsonBytes}
	server := httptest.NewServer(handler)
	defer server.Close()

	_, err := getPSLEntries(server.URL)
	if err == nil {
		t.Error("expected error from getPSLEntries with empty filtered results mockHandler. Got nil")
	}
}

func TestRenderData(t *testing.T) {
	entries := []*pslEntry{
		{
			ALabel:                  "ceepeeyou",
			ULabel:                  "ceepeeyou",
			DateOfContractSignature: "2099-06-13",
			RegistryOperator:        "@cpu's bargain gTLD emporium",
		},
		{
			ALabel:                  "cpu",
			ULabel:                  "ｃｐｕ",
			DateOfContractSignature: "2019-06-13",
		},
	}

	expectedList := `// ceepeeyou : 2099-06-13 @cpu's bargain gTLD emporium
ceepeeyou

// cpu : 2019-06-13
ｃｐｕ

`

	var buf bytes.Buffer
	if err := renderData(entries, io.Writer(&buf)); err != nil {
		t.Fatalf("unexpected error from renderData: %v", err)
	}

	rendered := buf.String()

	lines := strings.Split(rendered, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least two header lines in rendered data. "+
			"Found only %d lines", len(lines))
	}

	listContent := strings.Join(lines[3:], "\n")
	fmt.Printf("Got: \n%s\n", listContent)
	fmt.Printf("Expected: \n%s\n", expectedList)
	if listContent != expectedList {
		t.Errorf("expected rendered list content %q, got %q",
			expectedList, listContent)
	}
}
