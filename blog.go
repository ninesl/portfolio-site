package main

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	gitRoot   string
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
	return newBlog(os.DirFS(path), path)
}

func newBlog(blogFS fs.FS, gitRoot string) (*Blog, error) {
	b := &Blog{
		articles:  make(map[string]templ.Component),
		slugs:     make(map[string]templ.Component),
		ambiguous: make(map[string]bool),
		gitRoot:   gitRoot,
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
		if name == ".git" || name == ".github" {
			continue
		}
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
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		updated := info.ModTime()
		if b.gitRoot != "" {
			if added, ok := gitAdditionDate(b.gitRoot, path.Join(dir, name)); ok {
				updated = added
			}
		}
		component, err := convertMDToHTML(blogFS, path.Join(dir, base), updated)
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
		items = append(items, datedEntry{entry: pages.BlogEntry{Title: title, Path: articlePath}, updated: updated})
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

// Git doesn't track filesystem modification times. Follow renames back to the
// original commit and retain the author's timezone so late-night posts keep
// their original calendar day. Fall back to mtime outside a Git checkout.
func gitAdditionDate(gitRoot, filename string) (time.Time, bool) {
	output, err := exec.Command("git", "-C", gitRoot, "log", "--follow", "--format=%aI", "--", filename).Output()
	if err != nil {
		return time.Time{}, false
	}
	dates := strings.TrimSpace(string(output))
	last := dates
	if i := strings.LastIndexByte(dates, '\n'); i >= 0 {
		last = dates[i+1:]
	}
	date, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return time.Time{}, false
	}
	return date, true
}

func convertMDToHTML(blogFS fs.FS, blogName string, date time.Time) (templ.Component, error) {
	mdBytes, err := fs.ReadFile(blogFS, blogName+".md")
	if err != nil {
		return nil, err
	}

	mdBytes = append([]byte("# "+formatBlogName(path.Base(blogName))+"\n\n<p class=\"article-date\"><small><em>"+date.Format("January 2, 2006")+"</em></small></p>\n\n"), mdBytes...)

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
