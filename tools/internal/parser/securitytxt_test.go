package parser

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCheckSecurityTxt(t *testing.T) {
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		in   string
		want string // empty means valid
	}{
		{
			name: "valid",
			in:   "Contact: mailto:security@example.com\nExpires: 2027-01-01T00:00:00.000Z\n",
		},
		{
			name: "valid_with_comments_crlf_and_mixed_case",
			in:   "# Our security policy\r\n\r\ncontact: https://example.com/security\r\nEXPIRES: 2027-01-01T00:00:00Z\r\n",
		},
		{
			name: "valid_signed",
			in: `-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA256

Contact: mailto:security@example.com
Expires: 2027-01-01T00:00:00Z
-----BEGIN PGP SIGNATURE-----

iQIzBAEBCAAdFiEE
-----END PGP SIGNATURE-----
`,
		},
		{
			name: "missing_contact",
			in:   "Expires: 2027-01-01T00:00:00Z\n",
			want: "no Contact field",
		},
		{
			name: "missing_expires",
			in:   "Contact: mailto:security@example.com\n",
			want: "no Expires field",
		},
		{
			name: "expired",
			in:   "Contact: mailto:security@example.com\nExpires: 2026-01-01T00:00:00Z\n",
			want: "expired on 2026-01-01T00:00:00Z",
		},
		{
			name: "invalid_expires",
			in:   "Contact: mailto:security@example.com\nExpires: next year\n",
			want: `invalid Expires value "next year"`,
		},
		{
			name: "html_page",
			in:   "<html><body>Not found</body></html>\n",
			want: "no Contact field",
		},
		{
			name: "html_page_on_one_long_line",
			in:   "<html><body>" + strings.Repeat("x", 64<<10) + "</body></html>",
			want: "no Contact field",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSecurityTxt(tc.in, now)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Errorf("checkSecurityTxt() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateSecurityTxt(t *testing.T) {
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	const valid = "Contact: mailto:security@example.com\nExpires: 2027-01-01T00:00:00Z\n"

	type file struct {
		contentType string
		body        string
	}
	files := map[string]file{
		"good.example":     {"text/plain; charset=utf-8", valid},
		"plain.example":    {"text/plain", valid},
		"wildcard.example": {"text/plain; charset=utf-8", valid},
		"expired.example":  {"text/plain; charset=utf-8", "Contact: mailto:security@example.com\nExpires: 2026-01-01T00:00:00Z\n"},
		"html.example":     {"text/html; charset=utf-8", "<html><body>Welcome</body></html>"},
	}
	var (
		mu      sync.Mutex
		fetched []string
	)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		fetched = append(fetched, r.Host+r.URL.Path)
		mu.Unlock()
		f, ok := files[r.Host]
		if !ok || r.URL.Path != "/.well-known/security.txt" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", f.contentType)
		w.Write([]byte(f.body))
	}))
	defer server.Close()

	// Route every request to the test server, and verify its
	// certificate as the hostname it was issued for, while leaving
	// the requested host in the URL and Host header untouched.
	client := server.Client()
	transport := client.Transport.(*http.Transport)
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, server.Listener.Addr().String())
	}
	transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"

	in := list(
		section(1, 12, "PRIVATE DOMAINS",
			suffixes(2, 11, noInfo,
				suffix(3, "good.example"),
				suffix(4, "plain.example"),
				suffix(5, "missing.example"),
				suffix(6, "html.example"),
				suffix(7, "expired.example"),
				wildcard(8, 9, "wildcard.example"),
				markUnchanged(suffix(10, "unchanged.example")),
			),
		),
	)

	var got []string
	for _, err := range validateSecurityTxt(context.Background(), in, client, now) {
		got = append(got, err.Error())
	}
	want := []string{
		"line 6: suffix missing.example has no security.txt at https://missing.example/.well-known/security.txt: HTTP status 404",
		`line 7: suffix html.example has no security.txt at https://html.example/.well-known/security.txt: served as "text/html; charset=utf-8", not text/plain`,
		"line 8: suffix expired.example has an invalid security.txt at https://expired.example/.well-known/security.txt: expired on 2026-01-01T00:00:00Z",
	}
	checkDiff(t, "validateSecurityTxt errors", got, want)

	fetchedWildcard := false
	for _, f := range fetched {
		switch f {
		case "unchanged.example/.well-known/security.txt":
			t.Errorf("validateSecurityTxt fetched security.txt for unchanged suffix")
		case "wildcard.example/.well-known/security.txt":
			fetchedWildcard = true
		}
	}
	if !fetchedWildcard {
		t.Errorf("validateSecurityTxt did not fetch security.txt for wildcard base domain")
	}
}
