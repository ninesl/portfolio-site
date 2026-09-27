package main

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/a-h/templ"
	"github.com/ninesl/portfolio-site/pages"
)

func handleServeBlogPost(blog *Blog) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		article, err := blog.ArticleHTML(r.PathValue("article"))
		serveBlogArticle(w, r, article, err)
	}
}

func handleServeBlogSlug(blog *Blog) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		article, err := blog.SlugHTML(r.PathValue("slug"))
		serveBlogArticle(w, r, article, err)
	}
}

func serveBlogArticle(w http.ResponseWriter, r *http.Request, c templ.Component, err error) {
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			renderNotFound(w, r)
		} else {
			log.Printf("render article: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		renderComponent(pages.BlogArticleFragment(c), w, r)
		return
	}
	renderComponent(pages.Layout(c, *pageConfig, r.URL.Path), w, r)
}

func renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	content := pages.NotFound()
	if r.Header.Get("HX-Request") == "true" {
		renderComponent(content, w, r)
		return
	}
	renderComponent(pages.Layout(content, *pageConfig, r.URL.Path), w, r)
}

// Helper type for safety with components
type RawHTML string

type Blog struct {
	articles  map[string]templ.Component
	slugs     map[string]templ.Component
	ambiguous map[string]bool
	entries   []pages.BlogEntry
}

// used in main()
func initBlog(path string) *Blog {
	blog, err := NewBlog(path)
	if err != nil {
		log.Fatal(err)
	}
	return blog
}

func NewBlog(path string) (*Blog, error) {
	return newBlog(os.DirFS(path))
}

func newBlog(blogFS fs.FS) (*Blog, error) {
	b := &Blog{
		articles:  make(map[string]templ.Component),
		slugs:     make(map[string]templ.Component),
		ambiguous: make(map[string]bool),
	}
	entries, err := b.readEntries(blogFS, ".", "")
	if err != nil {
		return nil, err
	}
	b.entries = entries
	return b, nil
}

// ArticleHTML looks up a post by its formatted URL path, never its filename.
func (b *Blog) ArticleHTML(articlePath string) (templ.Component, error) {
	article, ok := b.articles[articlePath]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return article, nil
}

// SlugHTML resolves the short URL shown when an article is opened from the index.
func (b *Blog) SlugHTML(slug string) (templ.Component, error) {
	article, ok := b.slugs[slug]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return article, nil
}

func (b *Blog) Entries() []pages.BlogEntry {
	return b.entries
}

func (b *Blog) readEntries(blogFS fs.FS, dir, parent string) ([]pages.BlogEntry, error) {
	blogEntries, err := fs.ReadDir(blogFS, dir)
	if err != nil {
		return nil, err
	}
	type datedEntry struct {
		entry   pages.BlogEntry
		updated time.Time
	}
	items := make([]datedEntry, 0, len(blogEntries))
	for _, entry := range blogEntries {
		name := entry.Name()
		if entry.IsDir() {
			title := formatBlogName(name)
			children, err := b.readEntries(blogFS, path.Join(dir, name), path.Join(parent, title))
			if err != nil {
				return nil, err
			}
			items = append(items, datedEntry{entry: pages.BlogEntry{Title: title, Children: children, Folder: true}})
			continue
		}
		if path.Ext(name) != ".md" {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		title := formatBlogName(base)
		articlePath := path.Join(parent, title)
		if _, exists := b.articles[articlePath]; exists {
			return nil, errors.New("duplicate formatted blog path: " + articlePath)
		}
		component, err := convertMDToHTML(blogFS, path.Join(dir, base))
		if err != nil {
			return nil, err
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		b.articles[articlePath] = component
		slug := pages.ArticleSlug(title)
		if _, exists := b.slugs[slug]; exists {
			delete(b.slugs, slug)
			b.ambiguous[slug] = true
		} else if !b.ambiguous[slug] {
			b.slugs[slug] = component
		}
		items = append(items, datedEntry{entry: pages.BlogEntry{Title: title, Path: articlePath}, updated: info.ModTime()})
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.entry.Folder != b.entry.Folder {
			return !a.entry.Folder
		}
		if !a.entry.Folder && !a.updated.Equal(b.updated) {
			return a.updated.After(b.updated)
		}
		return a.entry.Title < b.entry.Title
	})
	entries := make([]pages.BlogEntry, len(items))
	for i, item := range items {
		entries[i] = item.entry
	}
	return entries, nil
}

func convertMDToHTML(blogFS fs.FS, blogName string) (templ.Component, error) {
	mdBytes, err := fs.ReadFile(blogFS, blogName+".md")
	if err != nil {
		return nil, err
	}
	fileInfo, err := fs.Stat(blogFS, blogName+".md")
	if err != nil {
		return nil, err
	}

	date := fileInfo.ModTime().Format("January 2, 2006")
	mdBytes = append([]byte("# "+formatBlogName(path.Base(blogName))+"\n\n<p class=\"article-date\"><small><em>"+date+"</em></small></p>\n\n"), mdBytes...)

	html, err := renderMarkdown(mdBytes)
	if err != nil {
		return nil, err
	}
	return templ.Raw(string(html)), nil
}

func formatBlogName(name string) string {
	var title strings.Builder
	letters := []rune(name)
	for i, r := range letters {
		if i > 0 && unicode.IsUpper(r) &&
			(unicode.IsLower(letters[i-1]) || unicode.IsDigit(letters[i-1]) ||
				(unicode.IsUpper(letters[i-1]) && i+1 < len(letters) && unicode.IsLower(letters[i+1]))) {
			title.WriteByte(' ')
		}
		title.WriteRune(r)
	}
	return title.String()
}
