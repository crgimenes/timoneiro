package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, home string, content string) {
	t.Helper()

	configDir := filepath.Join(home, ".config", "timoneiro")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	configFile := filepath.Join(configDir, "init.filo")
	if err := os.WriteFile(configFile, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestLoadConfigFromFilo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `(do
  (set Addr "127.0.0.1:9090")
  (set Timeout 45))`)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if cfg.Addr != "127.0.0.1:9090" {
		t.Fatalf("Addr = %q, want %q", cfg.Addr, "127.0.0.1:9090")
	}
	if cfg.Timeout != 45 {
		t.Fatalf("Timeout = %d, want 45", cfg.Timeout)
	}
}

func TestLoadConfigCreatesDefaultFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.Timeout != 30 {
		t.Fatalf("Timeout = %d, want 30", cfg.Timeout)
	}

	configFile := filepath.Join(home, ".config", "timoneiro", "init.filo")
	b, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	want := defaultConfigFilo(defaultConfig())
	if string(b) != want {
		t.Fatalf("created config = %q, want %q", string(b), want)
	}

	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config file mode = %v, want 0600", got)
	}
}

func TestConfigFilePathUsesHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := configFilePath()
	if err != nil {
		t.Fatalf("configFilePath: %v", err)
	}

	want := filepath.Join(home, ".config", "timoneiro", "init.filo")
	if got != want {
		t.Fatalf("configFilePath = %q, want %q", got, want)
	}
}

func TestLoadConfigRejectsEmptyAddr(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `(set Addr "")`)

	_, err := loadConfig()
	if err == nil {
		t.Fatal("loadConfig succeeded with empty Addr")
	}
}

func TestLoadConfigRejectsInvalidTimeout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `(set Timeout 0)`)

	_, err := loadConfig()
	if err == nil {
		t.Fatal("loadConfig succeeded with invalid Timeout")
	}
}

func TestRewriteArticleLinksOnlyChangesHref(t *testing.T) {
	in := `<p>https://example.com/a b</p><a href="https://example.com/a b">read</a>`
	want := `<p>https://example.com/a b</p><a href="/?q=https%3A%2F%2Fexample.com%2Fa+b">read</a>`

	got := rewriteArticleLinks(in)
	if got != want {
		t.Fatalf("rewriteArticleLinks = %q, want %q", got, want)
	}
}

func TestHandlerShowsURLFormWhenQueryMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler(&config{Addr: ":8080", Timeout: 30}, parseTemplate(html)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	for _, want := range []string{
		`<link rel="stylesheet" href="/assets/timoneiro.css">`,
		`<form class="url-form" method="get" action="/">`,
		`name="q"`,
		`type="url"`,
		`<button type="submit">Send</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body does not contain %q: %s", want, body)
		}
	}
}

func TestHandlerShowsURLFormWhenQueryEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?q=", nil)
	rec := httptest.NewRecorder()

	handler(&config{Addr: ":8080", Timeout: 30}, parseTemplate(html)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `name="q"`) {
		t.Fatalf("response body does not contain q input: %s", rec.Body.String())
	}
}
