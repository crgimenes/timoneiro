package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-shiori/go-readability"
)

func handler(w http.ResponseWriter, r *http.Request) {

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
	//fmt.Println(article.TextContent)

	//fmt.Fprintf(w, article.TextContent)
	fmt.Fprintf(w, article.Content)
}

func main() {

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)

	/*

		r := mux.NewRouter()

		r.HandleFunc("/{query}", func(w http.ResponseWriter, r *http.Request) {
			v := mux.Vars(r)
			q := v["query"]

			url := "https://" + q
			fmt.Println(url)

			article, err := readability.FromURL(url, 30*time.Second)
			if err != nil {
				log.Fatalf("failed to parse %s, %v\n", url, err)
			}

			fmt.Printf("URL     : %s\n", url)
			fmt.Printf("Title   : %s\n", article.Title)
			fmt.Printf("Author  : %s\n", article.Byline)
			fmt.Printf("Length  : %d\n", article.Length)
			fmt.Printf("Excerpt : %s\n", article.Excerpt)
			fmt.Printf("SiteName: %s\n", article.SiteName)
			fmt.Printf("Image   : %s\n", article.Image)
			fmt.Printf("Favicon : %s\n", article.Favicon)
			fmt.Println()
			fmt.Println(article.TextContent)

			fmt.Fprintf(w, q)
		}).Methods(http.MethodGet)

		http.ListenAndServe(":8080", r)
	*/
}
