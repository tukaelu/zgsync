package converter

import (
	"strconv"
	"strings"
	"testing"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"github.com/tukaelu/zgsync/internal/testutil"
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
	errorChecker := testutil.NewErrorChecker(t)
	asserter := testutil.NewAssertionHelper(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, err := c.ConvertToHTML(tc.markdown)
			errorChecker.ExpectNoError(err, "ConvertToHTML()")
			asserter.Equal(tc.expected, actualHTMLContent, "HTML content")
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
			if tc.expected != actualHTMLContent {
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
			if tc.expected != actualHTMLContent {
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
			if tc.expected != actualHTMLContent {
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
			if tc.expected != actualHTMLContent {
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
			if tc.expected != actualHTMLContent {
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
		name     string
		html     string
		expected string
	}{
		{
			name:     "malformed HTML",
			html:     "<p>Unclosed paragraph",
			expected: "Unclosed paragraph",
		},
		{
			name:     "nested unclosed tags",
			html:     "<div><p>Nested <strong>content",
			expected: ":::\nNested **content**\n:::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ConvertToMarkdown should handle malformed HTML gracefully
			result, err := converter.ConvertToMarkdown(tt.html)
			if err != nil {
				t.Errorf("ConvertToMarkdown() should handle malformed HTML gracefully, but got error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ConvertToMarkdown() = %q, want %q", result, tt.expected)
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

func TestConvertToMarkdown_PluckAttributes_DataFence(t *testing.T) {
	// data-fence must be silently skipped — it is an internal goldmark-fences artifact
	// that would produce incorrect output if preserved in the Markdown round-trip.
	node := &html.Node{
		Attr: []html.Attribute{
			{Key: "data-fence", Val: "0"},
			{Key: "id", Val: "section"},
		},
	}

	attrs := pluckAttributes(node)
	if len(attrs) != 1 || attrs[0] != "#section" {
		t.Errorf("expected [\"#section\"] (data-fence skipped), got %v", attrs)
	}
}

func TestConvertToMarkdown_PluckAttributes_MultipleClasses(t *testing.T) {
	node := &html.Node{
		Attr: []html.Attribute{
			{Key: "class", Val: "foo bar baz"},
		},
	}

	attrs := pluckAttributes(node)
	if len(attrs) != 1 || attrs[0] != ".foo .bar .baz" {
		t.Errorf("expected [\".foo .bar .baz\"], got %v", attrs)
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

func TestConvertToHTML_Table(t *testing.T) {
	testCases := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "basic table",
			markdown: "| Name | Age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |\n",
			expected: "<table>\n<thead>\n<tr>\n<th>Name</th>\n<th>Age</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>Alice</td>\n<td>30</td>\n</tr>\n<tr>\n<td>Bob</td>\n<td>25</td>\n</tr>\n</tbody>\n</table>\n",
		},
		{
			name:     "aligned table",
			markdown: "| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |\n",
			expected: "<table>\n<thead>\n<tr>\n<th style=\"text-align:left\">Left</th>\n<th style=\"text-align:center\">Center</th>\n<th style=\"text-align:right\">Right</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td style=\"text-align:left\">a</td>\n<td style=\"text-align:center\">b</td>\n<td style=\"text-align:right\">c</td>\n</tr>\n</tbody>\n</table>\n",
		},
	}

	c := NewConverter(false)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualHTMLContent, _ := c.ConvertToHTML(tc.markdown)
			if tc.expected != actualHTMLContent {
				t.Errorf("expected %s, got %s", tc.expected, actualHTMLContent)
			}
		})
	}
}

func TestConvertToMarkdown_Table(t *testing.T) {
	testCases := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "basic table",
			html:     "<table><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr><td>Alice</td><td>30</td></tr><tr><td>Bob</td><td>25</td></tr></tbody></table>",
			expected: "| Name | Age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |",
		},
		{
			name:     "aligned table",
			html:     "<table><thead><tr><th align=\"left\">Left</th><th align=\"center\">Center</th><th align=\"right\">Right</th></tr></thead><tbody><tr><td align=\"left\">a</td><td align=\"center\">b</td><td align=\"right\">c</td></tr></tbody></table>",
			expected: "| Left | Center | Right |\n| :-- | :-: | --: |\n| a | b | c |",
		},
	}

	c := NewConverter(false)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := c.ConvertToMarkdown(tc.html)
			if err != nil {
				t.Errorf("ConvertToMarkdown() failed: %v", err)
			}
			if result != tc.expected {
				t.Errorf("ConvertToMarkdown() = %q, want %q", result, tc.expected)
			}
		})
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

func TestConvertToMarkdown_ReplacementDivNilNode(t *testing.T) {
	content := "this is a test content"
	selection := &goquery.Selection{Nodes: []*html.Node{nil}}
	opt := &md.Options{}

	replaced := replacementDiv(content, selection, opt)

	if *replaced != content {
		t.Errorf("expected %q, got %q", content, *replaced)
	}
}

func TestConvertToMarkdown_ReplacementHeadingsNilNode(t *testing.T) {
	content := "heading test"
	selection := &goquery.Selection{Nodes: []*html.Node{nil}}
	opt := &md.Options{}

	replaced := replacementHeadings(content, selection, opt)

	if *replaced != content {
		t.Errorf("expected %q, got %q", content, *replaced)
	}
}

func TestConvertToMarkdown_ReplacementHeadingsInvalidLevel(t *testing.T) {
	content := "heading test"
	node := &html.Node{
		Data: "hx",
		Attr: []html.Attribute{},
	}
	selection := &goquery.Selection{Nodes: []*html.Node{node}}
	opt := &md.Options{}

	replaced := replacementHeadings(content, selection, opt)

	if *replaced != content {
		t.Errorf("expected %q, got %q", content, *replaced)
	}
}
