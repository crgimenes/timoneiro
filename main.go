package main

import (
	"embed"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"

	goconfig "crg.eti.br/go/config"
	_ "crg.eti.br/go/config/json"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-shiori/go-readability"
	"github.com/gosimple/slug"
)

type config struct {
	Addr    string `json:"addr" cfg:"addr" cfgDefault:":8080" cfgRequired:"true"`
	Timeout int64  `json:"timeout" cfg:"timeout" cfgDefault:"30" cfgRequired:"true"`
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

func getLinks(html string) []string {
	links := []string{}

	re := regexp.MustCompile(`<a href="(.*?)"`)
	for _, match := range re.FindAllStringSubmatch(html, -1) {
		links = append(links, match[1])
	}

	return links
}

func handler(cfg *config, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["q"]
		if !ok {
			log.Println("'q' is missing")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("'q' parameter is missing"))
			return
		}

		q := keys[0]

		format := r.URL.Query().Get("f")

		if format == "" {
			format = "html"
		}

		if format != "html" && format != "md" {
			log.Println("invalid format")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid format"))
			return
		}

		slugName := strings.ReplaceAll(q, "https://", "")
		slugName = strings.ReplaceAll(slugName, "http://", "")
		slugName = slug.Make(slugName)

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

		re := regexp.MustCompile(`<a href="(.*?)"`)
		for _, match := range re.FindAllStringSubmatch(article.Content, -1) {
			u := match[1]
			ue := url.QueryEscape(u)
			article.Content = strings.Replace(
				article.Content,
				u,
				fmt.Sprintf("http://localhost:8080/?q=%v", ue),
				-1)
			fmt.Println("link:", u)
		}

		data := articleData{
			URL:     q,
			MDURL:   fmt.Sprintf("https://crg.eti.br/timoneiro?q=%v&f=md", url.QueryEscape(q)),
			HTMLURL: fmt.Sprintf("https://crg.eti.br/timoneiro?q=%v&f=html", url.QueryEscape(q)),
			MDSN:    slugName + ".md",
			HTMLSN:  slugName + ".html",
			Title:   article.Title,
			Byline:  article.Byline,
			Excerpt: article.Excerpt,
			Content: article.Content,
		}

		tmpl := parseTemplate(html)

		var b strings.Builder

		err = tmpl.Execute(&b, data)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
			return
		}

		if format == "md" {
			converter := md.NewConverter("", true, nil)
			markdown, err := converter.ConvertString(b.String())
			if err != nil {
				log.Fatal(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
				return
			}

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte(markdown))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(b.String()))

		return
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
