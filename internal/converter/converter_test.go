package converter

import (
	"strconv"
	"strings"
	"testing"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func TestConvertToHTML(t *testing.T) {
	// TODO: implement this test
}

func TestConvertToHTML_Div(t *testing.T) {
	testCases := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "simple div",
			markdown: ":::{}\nthis is a test content\n:::\n",
			expected: "<div data-fence=\"0\">\n<p>this is a test content</p>\n</div>\n",
		},
		{
			name:     "div with attributes",
			markdown: ":::{#header .header}\nthis is a test content\n:::\n",
			expected: "<div data-fence=\"0\" id=\"header\" class=\"header\">\n<p>this is a test content</p>\n</div>\n",
		},
	}

	c := NewConverter()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, _ := c.ConvertToHTML(tc.markdown)
			if strings.Compare(tc.expected, actualHTMLContent) != 0 {
				t.Errorf("expected %s, got %s", tc.expected, actualHTMLContent)
			}
		})
	}
}

func TestConvertToHTML_Headings(t *testing.T) {
	testCases := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "h1",
			markdown: "# this is a test content",
			expected: "<h1>this is a test content</h1>\n",
		},
		{
			name:     "h2",
			markdown: "## this is a test content",
			expected: "<h2>this is a test content</h2>\n",
		},
		{
			name:     "h3",
			markdown: "### this is a test content",
			expected: "<h3>this is a test content</h3>\n",
		},
		{
			name:     "h4",
			markdown: "#### this is a test content",
			expected: "<h4>this is a test content</h4>\n",
		},
		{
			name:     "h5",
			markdown: "##### this is a test content",
			expected: "<h5>this is a test content</h5>\n",
		},
		{
			name:     "h6",
			markdown: "###### this is a test content",
			expected: "<h6>this is a test content</h6>\n",
		},
		{
			name:     "h1 with attributes",
			markdown: "# this is a test content {#header .header}",
			expected: "<h1 id=\"header\" class=\"header\">this is a test content</h1>\n",
		},
	}

	c := NewConverter()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, _ := c.ConvertToHTML(tc.markdown)
			if strings.Compare(tc.expected, actualHTMLContent) != 0 {
				t.Errorf("expected %s, got %s", tc.expected, actualHTMLContent)
			}
		})
	}
}

func TestConvertToMarkdown(t *testing.T) {
	// TODO: implement this test
}

func TestConvertToMarkdown_PluckAttributes(t *testing.T) {
	node := &html.Node{
		Attr: []html.Attribute{
			{Key: "id", Val: "header"},
			{Key: "class", Val: "header"},
			{Key: "data", Val: "header"},
		},
	}

	attrs := pluckAttributes(node)
	expected := []string{"#header", ".header", "data=header"}

	for i, attr := range attrs {
		if attr != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], attr)
		}
	}
}

func TestConvertToMarkdown_ReplacementDiv(t *testing.T) {
	content := "this is a test content"
	div := &html.Node{
		Data: "div",
		Attr: []html.Attribute{},
	}
	selection := &goquery.Selection{Nodes: []*html.Node{div}}
	opt := &md.Options{}

	expextedContent := ":::\n" + content + "\n:::\n\n"
	replaced := replacementDiv(content, selection, opt)

	if *replaced != expextedContent {
		t.Errorf("expected %s, got %s", expextedContent, *replaced)
	}
}

func TestConvertToMarkdown_ReplacementDivWithAttributes(t *testing.T) {
	content := "this is a test content"
	div := &html.Node{
		Data: "div",
		Attr: []html.Attribute{
			{Key: "id", Val: "header"},
			{Key: "class", Val: "header"},
			{Key: "data", Val: "header"},
		},
	}
	selection := &goquery.Selection{Nodes: []*html.Node{div}}
	opt := &md.Options{}

	expextedContent := ":::{#header .header data=header}\n" + content + "\n:::\n\n"
	replaced := replacementDiv(content, selection, opt)

	if *replaced != expextedContent {
		t.Errorf("expected %s, got %s", expextedContent, *replaced)
	}
}

func TestConvertToMarkdown_ReplacementHeadings(t *testing.T) {
	content := "heading test"
	headings := []string{"h1", "h2", "h3", "h4", "h5", "h6"}

	for _, heading := range headings {
		node := &html.Node{
			Data: heading,
			Attr: []html.Attribute{},
		}
		selection := &goquery.Selection{Nodes: []*html.Node{node}}
		opt := &md.Options{}

		level, _ := strconv.Atoi(node.Data[1:])
		prefix := strings.Repeat("#", level)

		expextedContent := prefix + " " + content + "\n"
		replaced := replacementHeadings(content, selection, opt)

		if *replaced != expextedContent {
			t.Errorf("expected %s, got %s", expextedContent, *replaced)
		}
	}
}

func TestConvertToMarkdown_ReplacementHeadingsWithAttributes(t *testing.T) {
	content := "heading test"
	headings := []string{"h1", "h2", "h3", "h4", "h5", "h6"}

	for _, heading := range headings {
		node := &html.Node{
			Data: heading,
			Attr: []html.Attribute{
				{Key: "id", Val: heading},
				{Key: "class", Val: heading},
				{Key: "data", Val: heading},
			},
		}
		selection := &goquery.Selection{Nodes: []*html.Node{node}}
		opt := &md.Options{}

		level, _ := strconv.Atoi(node.Data[1:])
		prefix := strings.Repeat("#", level)

		expextedContent := prefix + " " + content + " {#" + heading + " ." + heading + " data=" + heading + "}\n"
		replaced := replacementHeadings(content, selection, opt)

		if *replaced != expextedContent {
			t.Errorf("expected %s, got %s", expextedContent, *replaced)
		}
	}
}
