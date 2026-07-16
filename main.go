package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/a-h/templ"
	"github.com/joho/godotenv"
	"github.com/ninesl/portfolio-site/pages"
)

var (
	//go:embed assets
	embeddedFiles embed.FS

	////go:embed blog
	//embeddedBlog embed.FS
	assetHandler http.Handler

	pageConfig = &pages.LayoutConfig{
		Title: "Lance Nines - ninescoding",
	}
)

func init() {
	assets, err := fs.Sub(embeddedFiles, "assets")
	if err != nil {
		log.Fatal(err)
	}
	assetHandler = http.StripPrefix("/assets/", http.FileServerFS(assets))
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func addMiddleware(next http.Handler) http.Handler {
	return logging(next)
}

func renderComponent(comp templ.Component, w http.ResponseWriter, r *http.Request) {
	err := comp.Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func serveAsset(w http.ResponseWriter, r *http.Request) {
	assetHandler.ServeHTTP(w, r)
}

func getPort() int {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT is required")
	}

	parsedPort, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("invalid PORT %q: %v", port, err)
	}
	return parsedPort
}

func main() {
	blog := initBlog("./blog/")
	port := getPort()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/{path...}", serveAsset)
	mux.HandleFunc("GET /blog/{article}", func(w http.ResponseWriter, r *http.Request) {
		c, err := blog.ArticleHTML(r.PathValue("article"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("HX-Request") == "true" {
			renderComponent(c, w, r)
			return
		}
		renderComponent(pages.Layout(c, *pageConfig), w, r)
	})
	mux.HandleFunc("POST /count", func(w http.ResponseWriter, r *http.Request) {
		pageConfig.COUNT++
		renderComponent(pages.PersistCounter(pageConfig.COUNT, true), w, r)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		renderComponent(pages.Layout(pages.BlogHome(blog.ArticleTitles()), *pageConfig), w, r)
	})
	log.Println("listening on ", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), addMiddleware(mux)))
}
