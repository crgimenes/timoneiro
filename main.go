package main

import (
	"embed"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/crgimenes/filo"
	"github.com/gosimple/slug"
)

type config struct {
	Addr    string
	Timeout int
}

type articleData struct {
	URL     string
	MDURL   string
	MDSN    string
	HTMLURL string
	HTMLSN  string
	Title   string
	Byline  string
	Excerpt string
	Content string
}

//go:embed template.html
var html string

//go:embed assets
var assets embed.FS

var articleLinkRE = regexp.MustCompile(`<a href="([^"]*)"`)

func rewriteArticleLinks(content string) string {
	return articleLinkRE.ReplaceAllStringFunc(content, func(link string) string {
		match := articleLinkRE.FindStringSubmatch(link)
		if len(match) != 2 {
			return link
		}
		return fmt.Sprintf(`<a href="/?q=%s"`, url.QueryEscape(match[1]))
	})
}

func handler(cfg *config, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["q"]
		if !ok {
			log.Println("'q' is missing")
			http.Error(w, "'q' parameter is missing", http.StatusBadRequest)
			return
		}

		q := strings.TrimSpace(keys[0])
		if q == "" {
			log.Println("'q' is empty")
			http.Error(w, "'q' parameter is empty", http.StatusBadRequest)
			return
		}

		format := r.URL.Query().Get("f")

		if format == "" {
			format = "html"
		}

		if format != "html" && format != "md" {
			log.Println("invalid format")
			http.Error(w, "invalid format", http.StatusBadRequest)
			return
		}

		articleURL, err := url.Parse(q)
		if err != nil || articleURL.Scheme == "" || articleURL.Host == "" {
			log.Println("invalid article url")
			http.Error(w, "invalid article URL", http.StatusBadRequest)
			return
		}
		if articleURL.Scheme != "http" && articleURL.Scheme != "https" {
			log.Println("unsupported article url scheme")
			http.Error(w, "unsupported article URL scheme", http.StatusBadRequest)
			return
		}

		slugName := strings.ReplaceAll(q, "https://", "")
		slugName = strings.ReplaceAll(slugName, "http://", "")
		slugName = slug.Make(slugName)

		article, err := readability.FromURL(articleURL.String(), time.Duration(cfg.Timeout)*time.Second)
		if err != nil {
			log.Printf("failed to parse article: %v", err)
			http.Error(w, "failed to parse article", http.StatusBadGateway)
			return
		}

		var articleContent strings.Builder
		if err := article.RenderHTML(&articleContent); err != nil {
			log.Printf("failed to render article: %v", err)
			http.Error(w, "failed to render article", http.StatusBadGateway)
			return
		}

		data := articleData{
			URL:     q,
			MDURL:   fmt.Sprintf("https://crg.eti.br/timoneiro?q=%v&f=md", url.QueryEscape(q)),
			HTMLURL: fmt.Sprintf("https://crg.eti.br/timoneiro?q=%v&f=html", url.QueryEscape(q)),
			MDSN:    slugName + ".md",
			HTMLSN:  slugName + ".html",
			Title:   article.Title(),
			Byline:  article.Byline(),
			Excerpt: article.Excerpt(),
			Content: rewriteArticleLinks(articleContent.String()),
		}

		var b strings.Builder

		err = tmpl.Execute(&b, data)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if format == "md" {
			converter := md.NewConverter("", true, nil)
			markdown, err := converter.ConvertString(b.String())
			if err != nil {
				log.Println(err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			if _, err := io.WriteString(w, markdown); err != nil {
				log.Printf("write markdown response: %v", err)
			}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := io.WriteString(w, b.String()); err != nil {
			log.Printf("write html response: %v", err)
		}
	}
}

func writeResponse(w io.Writer, tmpl *template.Template, data articleData) {
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
	}
}

func parseTemplate(html string) *template.Template {
	tmpl := template.New("")

	_, err := tmpl.Parse(html)
	if err != nil {
		log.Fatalln(err)
	}

	return tmpl
}

func defaultConfig() config {
	return config{
		Addr:    ":8080",
		Timeout: 30,
	}
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "timoneiro")
	if err := os.MkdirAll(configPath, 0o700); err != nil {
		return "", fmt.Errorf("create config directory %s: %w", configPath, err)
	}

	return filepath.Join(configPath, "init.filo"), nil
}

func loadConfig() (*config, error) {
	cfg := defaultConfig()

	configFile, err := configFilePath()
	if err != nil {
		return nil, err
	}

	F := filo.New()
	defer F.Close()

	F.SetGlobal("Addr", cfg.Addr)
	F.SetGlobal("Timeout", cfg.Timeout)

	b, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", configFile, err)
	}

	if err := F.DoString(string(b)); err != nil {
		return nil, fmt.Errorf("execute config %s: %w", configFile, err)
	}

	cfg.Addr, err = F.GetString("Addr")
	if err != nil {
		return nil, err
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("Addr must not be empty")
	}

	cfg.Timeout, err = F.GetInt("Timeout")
	if err != nil {
		return nil, err
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("Timeout must be greater than zero")
	}

	return &cfg, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	tmpl := parseTemplate(html)

	mux := http.NewServeMux()

	mux.Handle("/assets/", http.FileServer(http.FS(assets)))
	mux.HandleFunc("/", handler(cfg, tmpl))

	s := &http.Server{
		Handler:        mux,
		Addr:           cfg.Addr,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("Listening on port %s\n", cfg.Addr)
	log.Fatal(s.ListenAndServe())

}
