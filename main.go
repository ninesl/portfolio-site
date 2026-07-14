package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gomarkdown/markdown"
	"github.com/ninesl/portfolio-site/pages"
)

var (
	//go:embed assets
	embeddedFiles embed.FS
	//go:embed blog
	embeddedBlog embed.FS

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

func renderBlogPost() string {
	md, err := embeddedBlog.ReadFile("blog/SetupPodmanEnabledPrivateVPSToHostSideProjects.md")
	if err != nil {
		log.Fatal(err)
	}
	renderer := newCustomizedRender()
	return string(markdown.ToHTML(md, nil, renderer))
}

func main() {
	blogHTML := renderBlogPost()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/{path...}", serveAsset)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		renderComponent(pages.Layout(pages.HomePage(blogHTML), *pageConfig), w, r)
	})
	mux.HandleFunc("POST /count", func(w http.ResponseWriter, r *http.Request) {
		pageConfig.COUNT++
		renderComponent(pages.PersistCounter(pageConfig.COUNT, true), w, r)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", addMiddleware(mux)))
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
