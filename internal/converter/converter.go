package converter

import (
	"bytes"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	md "github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
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

	html := md.NewConverter(
		md.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(
				table.WithCellPaddingBehavior(table.CellPaddingBehaviorMinimal),
			),
		),
		md.WithEscapeMode(md.EscapeModeDisabled),
	)
	html.Register.RendererFor("div", md.TagTypeBlock, renderDiv, md.PriorityEarly)
	for _, tag := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		html.Register.RendererFor(tag, md.TagTypeBlock, renderHeading, md.PriorityEarly)
	}

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

func renderDiv(ctx md.Context, w md.Writer, n *html.Node) md.RenderStatus {
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, n)

	attrs := pluckAttributes(n)
	styledDiv := ":::"
	if len(attrs) > 0 {
		styledDiv = styledDiv + "{" + strings.Join(attrs, " ") + "}"
	}
	styledDiv = styledDiv + "\n" + strings.TrimSpace(buf.String()) + "\n:::\n\n"

	_, _ = w.WriteString(styledDiv)
	return md.RenderSuccess
}

func renderHeading(ctx md.Context, w md.Writer, n *html.Node) md.RenderStatus {
	// n.Data is guaranteed to be "h1".."h6": RendererFor only invokes this
	// for nodes whose tag name matches what it was registered for.
	level, _ := strconv.Atoi(n.Data[1:])
	prefix := strings.Repeat("#", level)

	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, n)
	content := buf.String()

	attrs := pluckAttributes(n)
	if len(attrs) > 0 {
		content = content + " {" + strings.Join(attrs, " ") + "}"
	}

	_, _ = w.WriteString(prefix + " " + content + "\n")
	return md.RenderSuccess
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
