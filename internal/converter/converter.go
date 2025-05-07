package converter

import (
	"bytes"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/PuerkitoBio/goquery"
	fences "github.com/stefanfritsch/goldmark-fences"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	renderer "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type Converter interface {
	ConvertToHTML(markdown string) (string, error)
	ConvertToMarkdown(html string) (string, error)
}

type converterImpl struct {
	markdown goldmark.Markdown
	html     *md.Converter
}

func NewConverter(enableLinkTargetBlank bool) Converter {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			&fences.Extender{}, // TODO: will implement the output of the `div` tag ourselves.
		),
		goldmark.WithParserOptions(
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			renderer.WithHardWraps(),
			renderer.WithUnsafe(),
		),
	)

	if enableLinkTargetBlank {
		markdown.Parser().AddOptions(
			parser.WithASTTransformers(
				util.Prioritized(LinkTargetTransformer, 100),
			),
		)
	}

	html := md.NewConverter("", true, &md.Options{EscapeMode: "disabled", CodeBlockStyle: "fenced"})
	html.Use(plugin.Table())
	html.AddRules(
		md.Rule{
			Filter:      []string{"div"},
			Replacement: replacementDiv,
		},
		md.Rule{
			Filter:      []string{"h1", "h2", "h3", "h4", "h5", "h6"},
			Replacement: replacementHeadings,
		})

	return &converterImpl{markdown, html}
}

func (c *converterImpl) ConvertToHTML(markdown string) (string, error) {
	var buf bytes.Buffer
	err := c.markdown.Convert([]byte(markdown), &buf)
	return buf.String(), err
}

func (c *converterImpl) ConvertToMarkdown(html string) (string, error) {
	return c.html.ConvertString(html)
}

func pluckAttributes(node *html.Node) []string {
	var attrs []string
	for _, attr := range node.Attr {
		switch attr.Key {
		case "id":
			attrs = append(attrs, "#"+attr.Val)
		case "class":
			var classes []string
			for _, class := range strings.Split(attr.Val, " ") {
				classes = append(classes, "."+class)
			}
			attrs = append(attrs, strings.Join(classes, " "))
		case "data-fence":
			// data-fence attribute will be skipped as it affects stefanfritsch/goldmark-fences.
		default:
			attrs = append(attrs, attr.Key+"="+attr.Val)
		}
	}
	return attrs
}

func replacementDiv(content string, selec *goquery.Selection, opt *md.Options) *string {
	var node *html.Node
	if node = selec.Get(0); node == nil {
		return md.String(content)
	}
	attrs := pluckAttributes(node)

	styledDiv := ":::"
	if len(attrs) > 0 {
		styledDiv = styledDiv + "{" + strings.Join(attrs, " ") + "}"
	}
	styledDiv = styledDiv + "\n" + strings.TrimSpace(content) + "\n:::\n\n"

	return md.String(styledDiv)
}

func replacementHeadings(content string, selec *goquery.Selection, opt *md.Options) *string {
	var node *html.Node
	if node = selec.Get(0); node == nil {
		return md.String(content)
	}

	level, err := strconv.Atoi(node.Data[1:])
	if err != nil {
		return md.String(content)
	}
	prefix := strings.Repeat("#", level)

	attrs := pluckAttributes(node)
	if len(attrs) > 0 {
		content = content + " {" + strings.Join(attrs, " ") + "}"
	}

	return md.String(prefix + " " + content + "\n")
}

type linkTargetTransformer struct{}

var LinkTargetTransformer = &linkTargetTransformer{}

func (t *linkTargetTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if link, ok := node.(*ast.Link); ok && entering {
			url := string(link.Destination)

			// only if not an internal link apply attributes
			if !strings.HasPrefix(url, "#") && !strings.HasPrefix(url, "/#") {
				link.SetAttribute([]byte("target"), []byte("_blank"))
				link.SetAttribute([]byte("rel"), []byte("noopener noreferrer"))
			}
		}
		return ast.WalkContinue, nil
	})
}
