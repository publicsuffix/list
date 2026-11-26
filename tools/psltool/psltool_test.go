package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/publicsuffix/list/tools/internal/parser"
)

func TestExportCheckResultsToGHA(t *testing.T) {
	tests := []struct {
		name string
		errs []error
		want string
	}{
		{
			name: "no_errors",
			errs: nil,
			want: "security_txt=pass\ndns_validation=pass\n",
		},
		{
			name: "unrelated_error",
			errs: []error{errors.New("file needs reformatting")},
			want: "security_txt=pass\ndns_validation=pass\n",
		},
		{
			name: "missing_security_txt",
			errs: []error{errors.New("file needs reformatting"), parser.ErrMissingSecurityTxt{}},
			want: "security_txt=fail\ndns_validation=pass\n",
		},
		{
			name: "invalid_security_txt",
			errs: []error{parser.ErrInvalidSecurityTxt{}, errors.New("file needs reformatting")},
			want: "security_txt=fail\ndns_validation=pass\n",
		},
		{
			name: "missing_txt_record",
			errs: []error{errors.New("file needs reformatting"), parser.ErrMissingTXTRecord{}},
			want: "security_txt=pass\ndns_validation=fail\n",
		},
		{
			name: "txt_record_mismatch",
			errs: []error{parser.ErrTXTRecordMismatch{}},
			want: "security_txt=pass\ndns_validation=fail\n",
		},
		{
			name: "txt_record_multiple_prs",
			errs: []error{parser.ErrTXTRecordMultiplePRs{}},
			want: "security_txt=pass\ndns_validation=fail\n",
		},
		{
			name: "txt_check_failure",
			errs: []error{parser.ErrTXTCheckFailure{}},
			want: "security_txt=pass\ndns_validation=fail\n",
		},
		{
			name: "missing_security_txt_and_txt_record",
			errs: []error{parser.ErrMissingTXTRecord{}, parser.ErrMissingSecurityTxt{}},
			want: "security_txt=fail\ndns_validation=fail\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// GitHub Actions creates the file, and every step
			// output of the job is appended to it.
			const earlierOutput = "earlier=value\n"
			path := filepath.Join(t.TempDir(), "github_output")
			if err := os.WriteFile(path, []byte(earlierOutput), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GITHUB_OUTPUT", path)

			if err := exportCheckResultsToGHA(tc.errs); err != nil {
				t.Fatalf("exportCheckResultsToGHA() = %v, want no error", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if want := earlierOutput + tc.want; string(got) != want {
				t.Errorf("GITHUB_OUTPUT contains %q, want %q", got, want)
			}
		})
	}
}

func TestExportCheckResultsToGHAOutsideGitHubActions(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")
	// A relative path would be created here if anything was written.
	t.Chdir(t.TempDir())

	if err := exportCheckResultsToGHA([]error{parser.ErrMissingSecurityTxt{}}); err != nil {
		t.Errorf("exportCheckResultsToGHA() = %v, want no error", err)
	}

	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("exportCheckResultsToGHA() created %d files, want none", len(files))
	}
}
