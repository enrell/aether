//go:build windows

package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckRequiresWindowsAsset(t *testing.T) {
	for name, assets := range map[string]string{
		"linux only":   `[{"name": "aether-linux-amd64", "browser_download_url": "https://example.com/a"}]`,
		"with windows": `[{"name": "aether-windows-amd64.exe", "browser_download_url": "https://example.com/w"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"tag_name": "v9.0.0", "html_url": "https://example.com", "assets": ` + assets + `}`))
			}))
			defer server.Close()
			release, err := check(context.Background(), "3.0.0", server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			if want := name == "with windows"; release.UpdateAvailable != want {
				t.Errorf("UpdateAvailable = %v, want %v", release.UpdateAvailable, want)
			}
		})
	}
}
