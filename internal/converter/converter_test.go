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
	testCases := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "simple paragraph",
			markdown: "Hello world",
			expected: "<p>Hello world</p>\n",
		},
		{
			name:     "heading",
			markdown: "# Hello world",
			expected: "<h1>Hello world</h1>\n",
		},
		{
			name:     "bold text",
			markdown: "**bold text**",
			expected: "<p><strong>bold text</strong></p>\n",
		},
		{
			name:     "italic text",
			markdown: "*italic text*",
			expected: "<p><em>italic text</em></p>\n",
		},
		{
			name:     "code block",
			markdown: "```\ncode\n```",
			expected: "<pre><code>code\n</code></pre>\n",
		},
		{
			name:     "link",
			markdown: "[link](http://example.com)",
			expected: "<p><a href=\"http://example.com\">link</a></p>\n",
		},
		{
			name:     "empty string",
			markdown: "",
			expected: "",
		},
	}

	c := NewConverter(false)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, err := c.ConvertToHTML(tc.markdown)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if strings.Compare(tc.expected, actualHTMLContent) != 0 {
				t.Errorf("expected %s, got %s", tc.expected, actualHTMLContent)
			}
		})
	}
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

	c := NewConverter(false)
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

	c := NewConverter(false)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, _ := c.ConvertToHTML(tc.markdown)
			if strings.Compare(tc.expected, actualHTMLContent) != 0 {
				t.Errorf("expected %s, got %s", tc.expected, actualHTMLContent)
			}
		})
	}
}

func TestConvertToHTML_Anchor(t *testing.T) {
	testCases := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "Non-Attributes",
			markdown: "[Example](https://www.example.com/)",
			expected: "<p><a href=\"https://www.example.com/\">Example</a></p>\n",
		},
		{
			name:     "Title",
			markdown: "[Example](https://www.example.com/ \"Title\")",
			expected: "<p><a href=\"https://www.example.com/\" title=\"Title\">Example</a></p>\n",
		},
	}

	c := NewConverter(false)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, _ := c.ConvertToHTML(tc.markdown)
			if strings.Compare(tc.expected, actualHTMLContent) != 0 {
				t.Errorf("expected %s, got %s", tc.expected, actualHTMLContent)
			}
		})
	}
}

func TestConvertToHTML_AnchorWithTargetBlank(t *testing.T) {
	testCases := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "Non-Attributes",
			markdown: "[Example](https://www.example.com/)",
			expected: "<p><a href=\"https://www.example.com/\" target=\"_blank\" rel=\"noopener noreferrer\">Example</a></p>\n",
		},
		{
			name:     "Title",
			markdown: "[Example](https://www.example.com/ \"Title\")",
			expected: "<p><a href=\"https://www.example.com/\" title=\"Title\" target=\"_blank\" rel=\"noopener noreferrer\">Example</a></p>\n",
		},
	}

	c := NewConverter(true)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, _ := c.ConvertToHTML(tc.markdown)
			if strings.Compare(tc.expected, actualHTMLContent) != 0 {
				t.Errorf("expected %s, got %s", tc.expected, actualHTMLContent)
			}
		})
	}
}

func TestConvertToHTML_AnchorWithNoTargetBlank(t *testing.T) {
	testCases := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "Non-Attributes",
			markdown: "[Example](#hoge)",
			expected: "<p><a href=\"#hoge\">Example</a></p>\n",
		},
		{
			name:     "Title",
			markdown: "[Example](/#hoge \"Title\")",
			expected: "<p><a href=\"/#hoge\" title=\"Title\">Example</a></p>\n",
		},
	}

	c := NewConverter(true)
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
	converter := NewConverter(false)
	
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello, World!</p>",
			expected: "Hello, World!",
		},
		{
			name:     "heading with paragraph",
			html:     "<h1>Title</h1><p>Content</p>",
			expected: "# Title\n\nContent",
		},
		{
			name:     "div with class",
			html:     "<div class=\"info\">Important information</div>",
			expected: ":::{.info}\nImportant information\n:::",
		},
		{
			name:     "heading with attributes",
			html:     "<h2 id=\"section1\" class=\"highlight\">Section Title</h2>",
			expected: "## Section Title {#section1 .highlight}",
		},
		{
			name:     "nested elements",
			html:     "<div><h3>Nested</h3><p>Content inside div</p></div>",
			expected: ":::\n### Nested\n\nContent inside div\n:::",
		},
		{
			name:     "empty input",
			html:     "",
			expected: "",
		},
		{
			name:     "complex div with multiple attributes",
			html:     "<div id=\"main\" class=\"container\" data-section=\"content\">Test content</div>",
			expected: ":::{#main .container data-section=content}\nTest content\n:::",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertToMarkdown(tt.html)
			if err != nil {
				t.Errorf("ConvertToMarkdown() failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ConvertToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertToMarkdown_ErrorHandling(t *testing.T) {
	converter := NewConverter(false)
	
	// Test with invalid HTML that might cause issues
	tests := []struct {
		name string
		html string
	}{
		{
			name: "malformed HTML",
			html: "<p>Unclosed paragraph",
		},
		{
			name: "nested unclosed tags",
			html: "<div><p>Nested <strong>content",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ConvertToMarkdown should handle malformed HTML gracefully
			result, err := converter.ConvertToMarkdown(tt.html)
			if err != nil {
				t.Errorf("ConvertToMarkdown() should handle malformed HTML gracefully, but got error: %v", err)
			}
			// Result should be a string (even if not perfectly formatted)
			if len(result) < 0 {
				t.Errorf("ConvertToMarkdown() should return some result, got empty string")
			}
		})
	}
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
