package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gomarkdown/markdown"
	"github.com/joho/godotenv"
	"github.com/ninesl/portfolio-site/pages"
)

var (
	//go:embed assets
	embeddedFiles embed.FS

	////go:embed blog
	//embeddedBlog embed.FS

	blogFiles fs.FS = os.DirFS("./blog/")

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

func getBlogTitleNames() []string {
	blogEntries, err := fs.ReadDir(blogFiles, ".")
	if err != nil {
		log.Fatal(err)
	}
	blogTitles := make([]string, 0, len(blogEntries))
	for _, entry := range blogEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		blogTitles = append(blogTitles, strings.TrimSuffix(entry.Name(), ".md"))
	}
	return blogTitles
}

func convertMDToHTML(blogPath string) (string, error) {
	mdBytes, err := fs.ReadFile(blogFiles, blogPath+".md")
	if err != nil {
		return "", err
	}

	renderer := newCustomizedRender()
	return string(markdown.ToHTML(mdBytes, nil, renderer)), nil
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
	port := getPort()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/{path...}", serveAsset)
	mux.HandleFunc("GET /blog/{article}", func(w http.ResponseWriter, r *http.Request) {
		blogHTML, err := convertMDToHTML(r.PathValue("article"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		renderComponent(pages.Layout(pages.MarkdownHTML(blogHTML), *pageConfig), w, r)
	})
	mux.HandleFunc("POST /count", func(w http.ResponseWriter, r *http.Request) {
		pageConfig.COUNT++
		renderComponent(pages.PersistCounter(pageConfig.COUNT, true), w, r)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		renderComponent(pages.Layout(pages.BlogHome(getBlogTitleNames()), *pageConfig), w, r)
	})
	log.Println("listening on ", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), addMiddleware(mux)))
}
