package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/go-shiori/go-readability"
)

func handler(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["q"]
		if !ok {
			log.Println("'q' is missing")
			return
		}

		q := keys[0]

		article, err := readability.FromURL(q, 30*time.Second)
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

		writeResponse(w, tmpl, article)
	}
}

func writeResponse(w http.ResponseWriter, tmpl *template.Template, article readability.Article) {
	if err := tmpl.Execute(w, article); err != nil {
		log.Println(err)
	}
}

var (
	//go:embed template.html
	html string
)

func parseTemplate() *template.Template {
	tmpl := template.New("")
	if _, err := tmpl.Parse(html); err != nil {
		log.Fatalln(err)
	}
	return tmpl
}

func main() {
	tmpl := parseTemplate()
	http.HandleFunc("/favicon.ico", http.NotFound)
	http.HandleFunc("/", handler(tmpl))
	http.ListenAndServe(":8080", nil)
}
