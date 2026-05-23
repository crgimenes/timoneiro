package main

import (
	"os"
	"path/filepath"
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
