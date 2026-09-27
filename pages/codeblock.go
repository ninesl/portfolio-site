package pages

import (
	"bytes"
	"fmt"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

func codeBlockCSS() string {
	formatter := html.New(html.WithClasses(true), html.TabWidth(2))
	if formatter == nil {
		panic("couldn't create html formatter")
	}

	settings := DefaultSettings
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `@font-face{font-family:%q;src:url(%q) format("truetype");font-display:swap}body{font-family:%q,sans-serif}`, settings.TextFont, settings.TextFontURL, settings.TextFont)
	fmt.Fprintf(&buf, `@font-face{font-family:%q;src:url(%q) format("opentype");font-display:swap}.chroma,code{font-family:%q,monospace}`, settings.CodeFont, settings.CodeFontURL, settings.CodeFont)
	buf.WriteString(`pre.chroma{max-width:100%;padding:1rem 1.125rem;border:1px solid color-mix(in srgb,currentColor 18%,transparent);border-radius:.6rem;box-sizing:border-box;box-shadow:0 .3rem 1rem #0002;overflow-x:auto;white-space:pre-wrap;overflow-wrap:anywhere;tab-size:2;line-height:1.5}code.chroma{padding:.12em .35em;border:1px solid color-mix(in srgb,currentColor 18%,transparent);border-radius:.3em;box-decoration-break:clone;-webkit-box-decoration-break:clone;overflow-wrap:anywhere;font-size:.9em}pre.chroma code{padding:0;border:0;border-radius:0;font-size:inherit}`)
	style := styles.Get(settings.SyntaxTheme)
	if style == nil {
		panic(fmt.Sprintf("didn't find style %q", settings.SyntaxTheme))
	}
	if err := formatter.WriteCSS(&buf, style); err != nil {
		panic(err)
	}
	return buf.String()
}

func codeBlockStyleHTML() string {
	return "<style>" + codeBlockCSS() + "</style>"
}
