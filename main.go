package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/gosidekick/goconfig"
	_ "github.com/gosidekick/goconfig/json"
)

type config struct {
	Addr    string `json:"addr" cfg:"addr" cfgDefault:":8080" cfgRequired:"true"`
	Timeout int64  `json:"timeout" cfg:"timeout" cfgDefault:"30" cfgRequired:"true"`
}

type articleData struct {
	URL     string
	Title   string
	Byline  string
	Excerpt string
	Content string
}

//go:embed template.html
var html string // nolint

func handler(cfg *config, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["q"]
		if !ok {
			log.Println("'q' is missing")
			return
		}

		q := keys[0]

		article, err := readability.FromURL(q, time.Duration(cfg.Timeout)*time.Second)
		if err != nil {
			log.Fatalf("failed to parse %s, %v\n", q, err)
		}

		fmt.Printf("URL     : %s\n", q)
		fmt.Printf("Title   : %s\n", article.Title)
		fmt.Printf("Author  : %s\n", article.Byline)
		fmt.Printf("Length  : %d\n", article.Length)
		fmt.Printf("Excerpt : %s\n", article.Excerpt)
		fmt.Printf("SiteName: %s\n", article.SiteName)
		fmt.Printf("Image   : %s\n", article.Image)
		fmt.Printf("Favicon : %s\n", article.Favicon)
		fmt.Println()

		data := articleData{
			URL:     q,
			Title:   article.Title,
			Byline:  article.Byline,
			Excerpt: article.Excerpt,
			Content: article.Content,
		}

		writeResponse(w, tmpl, data)
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

func main() {
	cfg := &config{}
	goconfig.File = "timoneiro.json"

	err := goconfig.Parse(cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	tmpl := parseTemplate(html)

	http.HandleFunc("/favicon.ico", http.NotFound)
	http.HandleFunc("/", handler(cfg, tmpl))

	fmt.Println("listening on", cfg.Addr)

	err = http.ListenAndServe(cfg.Addr, nil)
	if err != nil {
		fmt.Println(err)
	}
}
