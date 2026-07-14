package pages

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

func codeBlockCSS() string {
	formatter := html.New(html.WithClasses(true), html.TabWidth(2))
	if formatter == nil {
		panic("couldn't create html formatter")
	}

	themeNames := []string{"tokyonight-night", "github-dark", "modus-vivendi"}

	var buf bytes.Buffer
	for _, styleName := range themeNames {
		style := styles.Get(styleName)
		if style == nil {
			panic(fmt.Sprintf("didn't find style '%s'", styleName))
		}

		var themeCSS bytes.Buffer
		if err := formatter.WriteCSS(&themeCSS, style); err != nil {
			panic(err)
		}
		buf.WriteString(strings.ReplaceAll(themeCSS.String(), ".chroma", `[data-syntax-theme="`+styleName+`"] .chroma`))
	}
	return buf.String()
}

func codeBlockStyleHTML() string {
	return "<style>" + codeBlockCSS() + "</style>"
}
