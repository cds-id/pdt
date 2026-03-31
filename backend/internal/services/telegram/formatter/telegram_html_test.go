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
			input:    "Hello <world> & friends",
			expected: "Hello &lt;world&gt; &amp; friends",
		},
		{
			name:     "bold",
			input:    "**bold**",
			expected: "<b>bold</b>",
		},
		{
			name:     "italic with asterisks",
			input:    "*italic*",
			expected: "<i>italic</i>",
		},
		{
			name:     "italic with underscores",
			input:    "_italic_",
			expected: "<i>italic</i>",
		},
		{
			name:     "strikethrough",
			input:    "~~deleted~~",
			expected: "<s>deleted</s>",
		},
		{
			name:     "inline code",
			input:    "`code`",
			expected: "<code>code</code>",
		},
		{
			name:     "link",
			input:    "[text](https://example.com)",
			expected: `<a href="https://example.com">text</a>`,
		},
		{
			name:     "nested bold and italic",
			input:    "***bold italic***",
			expected: "<i><b>bold italic</b></i>",
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
			input:    "---",
			expected: "————————————",
		},
		{
			name:     "nested list",
			input:    "- item one\n  - nested item\n- item two",
			expected: "• item one\n  • nested item\n• item two",
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
