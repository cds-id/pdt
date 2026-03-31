package formatter_test

import (
	"testing"

	"github.com/cds-id/pdt/backend/internal/services/telegram/formatter"
)

func TestToTelegramHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text with special chars",
			input:    `Hello <world> & "friends"`,
			expected: `Hello &lt;world&gt; &amp; "friends"`,
		},
		{
			name:     "bold",
			input:    "This is **bold** text",
			expected: "This is <b>bold</b> text",
		},
		{
			name:     "italic with asterisks",
			input:    "This is *italic* text",
			expected: "This is <i>italic</i> text",
		},
		{
			name:     "italic with underscores",
			input:    "This is _italic_ text",
			expected: "This is <i>italic</i> text",
		},
		{
			name:     "strikethrough",
			input:    "This is ~~deleted~~ text",
			expected: "This is <s>deleted</s> text",
		},
		{
			name:     "inline code",
			input:    "Use `fmt.Println` here",
			expected: "Use <code>fmt.Println</code> here",
		},
		{
			name:     "link",
			input:    "Visit [Google](https://google.com) now",
			expected: `Visit <a href="https://google.com">Google</a> now`,
		},
		{
			name:     "nested bold and italic",
			input:    "This is ***bold italic*** text",
			expected: "This is <i><b>bold italic</b></i> text",
		},
		{
			name:     "image to link",
			input:    "![alt text](https://example.com/img.png)",
			expected: `<a href="https://example.com/img.png">alt text</a>`,
		},
		{
			name:     "h1 heading uppercase",
			input:    "# Hello World",
			expected: "<b>HELLO WORLD</b>",
		},
		{
			name:     "h2 heading",
			input:    "## Subheading",
			expected: "<b>Subheading</b>",
		},
		{
			name:     "h3 heading",
			input:    "### H3 Title",
			expected: "<b>H3 Title</b>",
		},
		{
			name:     "fenced code block with language",
			input:    "```go\nfmt.Println(\"hello\")\n```",
			expected: "<pre><code class=\"language-go\">fmt.Println(\"hello\")\n</code></pre>",
		},
		{
			name:     "fenced code block no language",
			input:    "```\nsome code\n```",
			expected: "<pre><code>some code\n</code></pre>",
		},
		{
			name:     "unordered list",
			input:    "- item one\n- item two\n- item three",
			expected: "• item one\n• item two\n• item three",
		},
		{
			name:     "ordered list",
			input:    "1. first\n2. second\n3. third",
			expected: "1. first\n2. second\n3. third",
		},
		{
			name:     "blockquote",
			input:    "> some quoted text",
			expected: "<blockquote>some quoted text</blockquote>",
		},
		{
			name:     "horizontal rule",
			input:    "Above\n\n---\n\nBelow",
			expected: "Above\n\n————————————\n\nBelow",
		},
		{
			name:     "nested list",
			input:    "- item one\n  - nested item\n- item two",
			expected: "• item one\n  • nested item\n• item two",
		},
		{
			name:     "simple table",
			input:    "| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |",
			expected: "<pre>Name  | Age\n------+----\nAlice | 30\nBob   | 25</pre>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatter.ToTelegramHTML(tt.input)
			if got != tt.expected {
				t.Errorf("ToTelegramHTML(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.expected)
			}
		})
	}
}
