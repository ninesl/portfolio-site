package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/gomarkdown/markdown"
)

// Helper type for safety with components
type RawHTML string

type Blog struct {
	articles map[string]templ.Component
	blogFS   fs.FS
	mu       sync.RWMutex
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
	var (
		blogFiles = os.DirFS(path)
		titles    = getBlogTitleNames(blogFiles)
		articles  = make(map[string]templ.Component, len(titles))
	)

	for _, articleName := range titles {
		c, err := convertMDToHTML(blogFiles, articleName)
		if err != nil {
			return &Blog{}, err
		}
		articles[articleName] = c
	}

	return &Blog{
		articles: articles,
		blogFS:   blogFiles,
		mu:       sync.RWMutex{},
	}, nil
	//blogFiles:
}

// ArticleHTML returns the HTML templ component of givne articleName
//
// The article is read from the Blog's filesystem if it does not exist in the cache yet,
// and then prepares it for subsequent calls.
func (b *Blog) ArticleHTML(articleName string) (templ.Component, error) {
	b.mu.RLock()
	article, exists := b.articles[articleName]
	b.mu.RUnlock()

	if !exists {
		a, err := convertMDToHTML(b.blogFS, articleName)
		if err != nil {
			return nil, err
		}
		b.mu.Lock()
		b.articles[articleName] = a
		b.mu.Unlock()
	}

	return article, nil
}

func (b *Blog) ArticleTitles() []string {
	articleTitles := make([]string, 0, len(b.articles))

	b.mu.RLock()
	for title := range b.articles {
		articleTitles = append(articleTitles, title)
	}
	b.mu.RUnlock()

	// FIXME: sort?
	return articleTitles
}

func getBlogTitleNames(blogFS fs.FS) []string {
	blogEntries, err := fs.ReadDir(blogFS, ".")
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

func convertMDToHTML(blogFS fs.FS, blogName string) (templ.Component, error) {
	mdBytes, err := fs.ReadFile(blogFS, blogName+".md")
	if err != nil {
		return nil, err
	}

	renderer := newCustomizedRender()
	return templ.Raw(string(markdown.ToHTML(mdBytes, nil, renderer))), nil
}
