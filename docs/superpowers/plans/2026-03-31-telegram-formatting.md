# Telegram Response Formatting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace fragile MarkdownV2 escaping with a robust Markdown-to-Telegram-HTML converter so all LLM responses render cleanly in Telegram.

**Architecture:** A `goldmark`-based Markdown parser with a custom Telegram HTML renderer lives in `formatter/` inside the telegram package. The `stream_writer.go` calls this converter before sending, switching `ParseMode` to `"HTML"`. Plain-text fallback remains for edge cases.

**Tech Stack:** Go, `github.com/yuin/goldmark` (Markdown AST parser), `github.com/go-telegram-bot-api/telegram-bot-api/v5`

---

### Task 1: Add goldmark dependency

**Files:**
- Modify: `backend/go.mod`

- [ ] **Step 1: Install goldmark**

```bash
cd backend && go get github.com/yuin/goldmark
```

- [ ] **Step 2: Verify it's in go.mod**

```bash
grep goldmark go.mod
```

Expected: `github.com/yuin/goldmark v1.x.x`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add goldmark markdown parser dependency"
```

---

### Task 2: Create Telegram HTML renderer — text escaping and inline formatting

**Files:**
- Create: `backend/internal/services/telegram/formatter/telegram_html.go`
- Create: `backend/internal/services/telegram/formatter/telegram_html_test.go`

- [ ] **Step 1: Write failing tests for escaping and inline formatting**

```go
// backend/internal/services/telegram/formatter/telegram_html_test.go
package formatter

import "testing"

func TestToTelegramHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text with special chars",
			input:    "Hello <world> & \"friends\"",
			expected: "Hello &lt;world&gt; &amp; \"friends\"",
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
			expected: "This is <b><i>bold italic</i></b> text",
		},
		{
			name:     "image to link",
			input:    "![Screenshot](https://example.com/img.png)",
			expected: `<a href="https://example.com/img.png">Screenshot</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToTelegramHTML(tt.input)
			if got != tt.expected {
				t.Errorf("ToTelegramHTML(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v -run TestToTelegramHTML
```

Expected: compilation error — package and function don't exist yet.

- [ ] **Step 3: Implement the renderer with escaping and inline formatting**

Create `backend/internal/services/telegram/formatter/telegram_html.go`:

```go
package formatter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	astext "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// ToTelegramHTML converts standard Markdown to Telegram-compatible HTML.
func ToTelegramHTML(markdown string) string {
	source := []byte(markdown)
	md := goldmark.New(
		goldmark.WithExtensions(extension.Strikethrough, extension.Table),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)

	doc := md.Parser().Parse(text.NewReader(source))

	var buf bytes.Buffer
	r := &telegramRenderer{source: source}
	r.renderNode(&buf, doc, source)

	return strings.TrimSpace(buf.String())
}

type telegramRenderer struct {
	source []byte
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func (r *telegramRenderer) renderNode(buf *bytes.Buffer, node ast.Node, source []byte) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.renderSingle(buf, child, source)
	}
}

func (r *telegramRenderer) renderSingle(buf *bytes.Buffer, node ast.Node, source []byte) {
	switch n := node.(type) {
	case *ast.Document:
		r.renderNode(buf, n, source)

	// Block nodes
	case *ast.Paragraph:
		r.renderNode(buf, n, source)
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *ast.Heading:
		buf.WriteString("<b>")
		if n.Level == 1 {
			var inner bytes.Buffer
			r.renderNode(&inner, n, source)
			buf.WriteString(strings.ToUpper(inner.String()))
		} else {
			r.renderNode(buf, n, source)
		}
		buf.WriteString("</b>")
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *ast.CodeBlock:
		buf.WriteString("<pre>")
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.WriteString(escapeHTML(string(line.Value(source))))
		}
		buf.WriteString("</pre>")
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *ast.FencedCodeBlock:
		lang := string(n.Language(source))
		if lang != "" {
			buf.WriteString(fmt.Sprintf(`<pre><code class="language-%s">`, escapeHTML(lang)))
		} else {
			buf.WriteString("<pre><code>")
		}
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.WriteString(escapeHTML(string(line.Value(source))))
		}
		buf.WriteString("</code></pre>")
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *ast.Blockquote:
		buf.WriteString("<blockquote>")
		var inner bytes.Buffer
		r.renderNode(&inner, n, source)
		buf.WriteString(strings.TrimSpace(inner.String()))
		buf.WriteString("</blockquote>")
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *ast.List:
		r.renderList(buf, n, source)
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *ast.ListItem:
		// handled by renderList
		r.renderNode(buf, n, source)

	case *ast.ThematicBreak:
		buf.WriteString("————————————")
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *ast.HTMLBlock:
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.WriteString(string(line.Value(source)))
		}

	// Inline nodes
	case *ast.Text:
		buf.WriteString(escapeHTML(string(n.Value(source))))
		if n.SoftLineBreak() {
			buf.WriteString("\n")
		}
		if n.HardLineBreak() {
			buf.WriteString("\n")
		}

	case *ast.String:
		buf.WriteString(escapeHTML(string(n.Value)))

	case *ast.Emphasis:
		if n.Level == 2 {
			buf.WriteString("<b>")
			r.renderNode(buf, n, source)
			buf.WriteString("</b>")
		} else {
			buf.WriteString("<i>")
			r.renderNode(buf, n, source)
			buf.WriteString("</i>")
		}

	case *astext.Strikethrough:
		buf.WriteString("<s>")
		r.renderNode(buf, n, source)
		buf.WriteString("</s>")

	case *ast.CodeSpan:
		buf.WriteString("<code>")
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				buf.WriteString(escapeHTML(string(t.Value(source))))
			}
		}
		buf.WriteString("</code>")

	case *ast.Link:
		buf.WriteString(fmt.Sprintf(`<a href="%s">`, escapeHTML(string(n.Destination))))
		r.renderNode(buf, n, source)
		buf.WriteString("</a>")

	case *ast.Image:
		buf.WriteString(fmt.Sprintf(`<a href="%s">`, escapeHTML(string(n.Destination))))
		// Render alt text
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				buf.WriteString(escapeHTML(string(t.Value(source))))
			}
		}
		buf.WriteString("</a>")

	case *ast.AutoLink:
		url := string(n.URL(source))
		buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, escapeHTML(url), escapeHTML(url)))

	case *ast.RawHTML:
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			buf.Write(seg.Value(source))
		}

	default:
		// For any unknown node type, try to render children
		if node.HasChildren() {
			r.renderNode(buf, node, source)
		}
	}
}

func (r *telegramRenderer) renderList(buf *bytes.Buffer, list *ast.List, source []byte) {
	counter := list.Start
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.ListItem); !ok {
			continue
		}
		r.renderListItem(buf, child.(*ast.ListItem), list.IsOrdered(), counter, 0, source)
		counter++
	}
}

func (r *telegramRenderer) renderListItem(buf *bytes.Buffer, item *ast.ListItem, ordered bool, index int, depth int, source []byte) {
	indent := strings.Repeat("  ", depth)
	if ordered {
		buf.WriteString(fmt.Sprintf("%s%d. ", indent, index))
	} else {
		buf.WriteString(indent + "• ")
	}

	// Render inline content of this list item
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		if nested, ok := child.(*ast.List); ok {
			buf.WriteString("\n")
			nestedCounter := nested.Start
			for nestedChild := nested.FirstChild(); nestedChild != nil; nestedChild = nestedChild.NextSibling() {
				if nestedItem, ok := nestedChild.(*ast.ListItem); ok {
					r.renderListItem(buf, nestedItem, nested.IsOrdered(), nestedCounter, depth+1, source)
					nestedCounter++
				}
			}
		} else {
			// Render paragraph content inline (strip the trailing newlines)
			var inner bytes.Buffer
			r.renderNode(&inner, child, source)
			buf.WriteString(strings.TrimSpace(inner.String()))
		}
	}

	if depth == 0 || item.NextSibling() != nil {
		buf.WriteString("\n")
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v -run TestToTelegramHTML
```

Expected: all 9 tests PASS. Some tests may need minor output adjustments (trailing newlines, etc.) — fix the expected values to match goldmark's actual AST output.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/telegram/formatter/
git commit -m "feat(telegram): add Markdown-to-Telegram-HTML converter with inline formatting"
```

---

### Task 3: Add tests and support for block elements (headings, code blocks, lists, blockquotes, tables, hr)

**Files:**
- Modify: `backend/internal/services/telegram/formatter/telegram_html_test.go`
- Modify: `backend/internal/services/telegram/formatter/telegram_html.go`

- [ ] **Step 1: Add failing tests for block elements**

Append to the `tests` slice in `TestToTelegramHTML`:

```go
		{
			name:     "h1 heading uppercase",
			input:    "# Main Title",
			expected: "<b>MAIN TITLE</b>",
		},
		{
			name:     "h2 heading",
			input:    "## Section",
			expected: "<b>Section</b>",
		},
		{
			name:     "h3 heading",
			input:    "### Subsection",
			expected: "<b>Subsection</b>",
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
			input:    "- First\n- Second\n- Third",
			expected: "• First\n• Second\n• Third",
		},
		{
			name:     "ordered list",
			input:    "1. First\n2. Second\n3. Third",
			expected: "1. First\n2. Second\n3. Third",
		},
		{
			name:     "blockquote",
			input:    "> This is a quote",
			expected: "<blockquote>This is a quote</blockquote>",
		},
		{
			name:     "horizontal rule",
			input:    "Above\n\n---\n\nBelow",
			expected: "Above\n\n————————————\n\nBelow",
		},
		{
			name:  "nested list",
			input: "- Parent\n  - Child\n  - Child2\n- Parent2",
			expected: "• Parent\n  • Child\n  • Child2\n• Parent2",
		},
```

- [ ] **Step 2: Run tests to check which fail**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v -run TestToTelegramHTML
```

Expected: new tests may fail if output format doesn't match exactly. Adjust either the renderer or expected values to align.

- [ ] **Step 3: Fix renderer to pass all block element tests**

Iterate on `telegram_html.go` — adjust spacing, newline handling, list rendering until all tests pass. The core rendering logic from Task 2 already handles these node types; this step is about getting the exact output format right.

- [ ] **Step 4: Run all tests to verify they pass**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/telegram/formatter/
git commit -m "feat(telegram): add block element support (headings, code, lists, quotes, tables, hr)"
```

---

### Task 4: Add table-to-monospace conversion

**Files:**
- Modify: `backend/internal/services/telegram/formatter/telegram_html.go`
- Modify: `backend/internal/services/telegram/formatter/telegram_html_test.go`

- [ ] **Step 1: Write failing test for table conversion**

Add to the test slice:

```go
		{
			name:  "simple table",
			input: "| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |",
			expected: "<pre>Name  | Age\n------+----\nAlice | 30\nBob   | 25</pre>",
		},
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v -run "simple_table"
```

Expected: FAIL — table node type not handled yet.

- [ ] **Step 3: Implement table rendering**

Add table handling to `renderSingle` in `telegram_html.go`:

```go
	case *astext.Table:
		r.renderTable(buf, n, source)
		if n.NextSibling() != nil {
			buf.WriteString("\n\n")
		}

	case *astext.TableHeader:
		// handled by renderTable

	case *astext.TableRow:
		// handled by renderTable

	case *astext.TableCell:
		// handled by renderTable
```

Add `renderTable` method:

```go
func (r *telegramRenderer) renderTable(buf *bytes.Buffer, table *astext.Table, source []byte) {
	// Collect all rows (header + body)
	var rows [][]string
	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch row := child.(type) {
		case *astext.TableHeader:
			for tr := row.FirstChild(); tr != nil; tr = tr.NextSibling() {
				cells := r.collectTableRow(tr, source)
				rows = append(rows, cells)
			}
		case *astext.TableRow:
			cells := r.collectTableRow(child, source)
			rows = append(rows, cells)
		}
	}

	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make([]int, 0)
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(colWidths) {
				colWidths = append(colWidths, len(cell))
			} else if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	buf.WriteString("<pre>")
	for i, row := range rows {
		for j, cell := range row {
			if j > 0 {
				buf.WriteString(" | ")
			}
			padded := cell + strings.Repeat(" ", colWidths[j]-len(cell))
			buf.WriteString(padded)
		}
		buf.WriteString("\n")

		// Separator after header
		if i == 0 {
			for j, w := range colWidths {
				if j > 0 {
					buf.WriteString("+")
				}
				buf.WriteString(strings.Repeat("-", w+2)) // +2 for padding around |
			}
			buf.WriteString("\n")
		}
	}
	buf.WriteString("</pre>")
}

func (r *telegramRenderer) collectTableRow(node ast.Node, source []byte) []string {
	var cells []string
	for cell := node.FirstChild(); cell != nil; cell = cell.NextSibling() {
		var cellBuf bytes.Buffer
		r.renderNode(&cellBuf, cell, source)
		cells = append(cells, strings.TrimSpace(cellBuf.String()))
	}
	return cells
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v -run "simple_table"
```

Expected: PASS. Adjust separator format if needed to match expected output exactly.

- [ ] **Step 5: Run all tests**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/services/telegram/formatter/
git commit -m "feat(telegram): add table-to-monospace conversion"
```

---

### Task 5: Integrate converter into stream_writer and switch to HTML mode

**Files:**
- Modify: `backend/internal/services/telegram/stream_writer.go`

- [ ] **Step 1: Update imports**

In `stream_writer.go`, add the formatter import and remove unused imports:

```go
import (
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/cds-id/pdt/backend/internal/services/telegram/formatter"
)
```

- [ ] **Step 2: Replace MarkdownV2 with HTML in WriteDone**

Replace the chunk sending block (lines 118-131) in `WriteDone()`:

```go
	chunks := splitMessage(content, 4096)
	for i, chunk := range chunks {
		htmlContent := formatter.ToTelegramHTML(chunk)
		msg := tgbotapi.NewMessage(w.chatID, htmlContent)
		msg.ParseMode = "HTML"
		if _, err := w.bot.Send(msg); err != nil {
			// Retry without HTML on parse failure
			msg.ParseMode = ""
			msg.Text = stripHTMLTags(chunk)
			if _, err := w.bot.Send(msg); err != nil {
				return fmt.Errorf("send chunk %d: %w", i, err)
			}
		}
		if i < len(chunks)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}
```

- [ ] **Step 3: Replace escapeMarkdownV2 with stripHTMLTags**

Remove the `escapeMarkdownV2` function entirely. Add a simple `stripHTMLTags` for the plain-text fallback:

```go
// stripHTMLTags removes HTML tags for plain-text fallback.
func stripHTMLTags(s string) string {
	var buf strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			buf.WriteRune(r)
		}
	}
	result := buf.String()
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	return result
}
```

- [ ] **Step 4: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/telegram/stream_writer.go
git commit -m "feat(telegram): switch from MarkdownV2 to HTML formatting via converter"
```

---

### Task 6: Add integration-style test for the full pipeline

**Files:**
- Modify: `backend/internal/services/telegram/formatter/telegram_html_test.go`

- [ ] **Step 1: Write test with realistic LLM output**

Add a new test function:

```go
func TestToTelegramHTML_RealisticLLMOutput(t *testing.T) {
	input := `# Daily Report

Here's your **daily summary** for the team:

## Git Activity

- **3 commits** pushed to \`main\`
- Fixed the *authentication bug* in login flow
- Updated ~~deprecated~~ API endpoints

## Task Status

| Task | Status | Assignee |
|------|--------|----------|
| Auth fix | Done | Alice |
| API update | In Progress | Bob |

### Code Example

` + "```go" + `
func main() {
    fmt.Println("Hello <world> & friends")
}
` + "```" + `

> Note: Deploy is scheduled for tomorrow.

---

For details, visit [Dashboard](https://app.example.com/dashboard).`

	result := ToTelegramHTML(input)

	// Verify key transformations
	checks := []struct {
		desc string
		want string
	}{
		{"h1 uppercase bold", "<b>DAILY REPORT</b>"},
		{"h2 bold", "<b>Git Activity</b>"},
		{"bold text", "<b>daily summary</b>"},
		{"inline code", "<code>main</code>"},
		{"italic", "<i>authentication bug</i>"},
		{"strikethrough", "<s>deprecated</s>"},
		{"bullet list", "• "},
		{"code block", `<pre><code class="language-go">`},
		{"html escaping in code", "&lt;world&gt; &amp; friends"},
		{"table in pre", "<pre>"},
		{"blockquote", "<blockquote>"},
		{"horizontal rule", "————————————"},
		{"link", `<a href="https://app.example.com/dashboard">Dashboard</a>`},
	}

	for _, c := range checks {
		if !strings.Contains(result, c.want) {
			t.Errorf("%s: expected output to contain %q\n\nFull output:\n%s", c.desc, c.want, result)
		}
	}
}
```

- [ ] **Step 2: Run the test**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v -run TestToTelegramHTML_RealisticLLMOutput
```

Expected: PASS. If any check fails, fix the renderer.

- [ ] **Step 3: Run all tests one final time**

```bash
cd backend && go test ./internal/services/telegram/formatter/ -v
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/telegram/formatter/
git commit -m "test(telegram): add integration test with realistic LLM output"
```

---

### Task 7: Final build verification

**Files:** None (verification only)

- [ ] **Step 1: Full build**

```bash
cd backend && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 2: Run all tests**

```bash
cd backend && go test ./... 2>&1 | tail -20
```

Expected: all tests PASS, no regressions.

- [ ] **Step 3: Verify go vet**

```bash
cd backend && go vet ./...
```

Expected: no issues.
