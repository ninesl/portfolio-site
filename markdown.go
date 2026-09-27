package main

import (
	"fmt"
	"io"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/ninesl/portfolio-site/pages"
)

var (
	htmlFormatter   *html.Formatter
	inlineFormatter *html.Formatter
	highlightStyle  *chroma.Style
)

func init() {
	htmlFormatter = html.New(html.WithClasses(true), html.TabWidth(2))
	inlineFormatter = html.New(html.WithClasses(true), html.InlineCode(true))
	if htmlFormatter == nil {
		panic("couldn't create html formatter")
	}
	styleName := pages.DefaultSettings.SyntaxTheme
	highlightStyle = styles.Get(styleName)
	if highlightStyle == nil {
		panic(fmt.Sprintf("didn't find style '%s'", styleName))
	}
}

// based on https://github.com/alecthomas/chroma/blob/master/quick/quick.go
func codeHighlightHTML(w io.Writer, source, lang string, formatter *html.Formatter) error {
	l := lexers.Get(lang)
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return formatter.Format(w, highlightStyle, it)
}

// Code nodes are leaves, so entering is ignored. It is useful for container
// features such as wrapping a blockquote in <aside>...</aside>, adding an
// anchor inside a heading, or rendering spoilers as <details>...</details>.
// CSS can style existing markup, but it cannot create those HTML structures.
func renderMarkdown(source []byte) ([]byte, error) {
	var renderErr error
	hook := func(w io.Writer, node ast.Node, _ bool) (ast.WalkStatus, bool) {
		if code, ok := node.(*ast.CodeBlock); ok {
			lang := string(code.Info)
			if lang == "" {
				lang = "text"
			}
			renderErr = codeHighlightHTML(w, string(code.Literal), lang, htmlFormatter)
		} else if code, ok := node.(*ast.Code); ok {
			lexer := lexers.Analyse(string(code.Literal))
			if lexer == nil {
				lexer = lexers.Fallback
			}
			it, err := chroma.Coalesce(lexer).Tokenise(nil, string(code.Literal))
			if err != nil {
				renderErr = err
			} else {
				renderErr = inlineFormatter.Format(w, highlightStyle, it)
			}
		} else {
			return ast.GoToNext, false
		}
		if renderErr != nil {
			return ast.Terminate, true
		}
		return ast.GoToNext, true
	}
	renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{
		Flags:          mdhtml.CommonFlags | mdhtml.HrefTargetBlank,
		RenderNodeHook: hook,
	})
	mdParser := parser.NewWithExtensions(parser.CommonExtensions | parser.NoEmptyLineBeforeBlock)
	return markdown.ToHTML(source, mdParser, renderer), renderErr
}
