package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// ToTelegramHTML converts standard Markdown to Telegram-compatible HTML.
func ToTelegramHTML(markdown string) string {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Strikethrough,
			extension.Table,
		),
	)

	src := []byte(markdown)
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	var buf bytes.Buffer
	walkNode(&buf, doc, src, 0)
	return strings.TrimSpace(buf.String())
}

// bufEndsWith checks whether buf ends with the given suffix without converting
// the entire buffer to a string (O(len(suffix)) instead of O(n)).
func bufEndsWith(buf *bytes.Buffer, suffix string) bool {
	b := buf.Bytes()
	s := []byte(suffix)
	if len(b) < len(s) {
		return false
	}
	return bytes.Equal(b[len(b)-len(s):], s)
}

// ensureDoubleNewline adds the necessary newlines so that the next block starts
// after a blank line, without ever writing more than two consecutive newlines.
func ensureDoubleNewline(buf *bytes.Buffer) {
	if buf.Len() == 0 {
		return
	}
	if !bufEndsWith(buf, "\n\n") {
		if bufEndsWith(buf, "\n") {
			buf.WriteString("\n")
		} else {
			buf.WriteString("\n\n")
		}
	}
}

// escapeHTML escapes only the characters that Telegram HTML requires escaping.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// walkNode recursively walks an AST node and writes Telegram HTML to buf.
// depth tracks list nesting level.
func walkNode(buf *bytes.Buffer, node ast.Node, src []byte, depth int) {
	switch n := node.(type) {
	case *ast.Document:
		writeChildren(buf, n, src, depth)

	case *ast.Paragraph:
		writeParagraph(buf, n, src, depth)

	case *ast.Heading:
		writeHeading(buf, n, src, depth)

	case *ast.FencedCodeBlock:
		writeFencedCode(buf, n, src)

	case *ast.CodeBlock:
		writeCodeBlock(buf, n, src)

	case *ast.Blockquote:
		writeBlockquote(buf, n, src, depth)

	case *ast.List:
		writeList(buf, n, src, depth)

	case *ast.ThematicBreak:
		writeThematicBreak(buf, n)

	case *ast.HTMLBlock:
		// skip raw HTML blocks

	case *ast.Text:
		writeText(buf, n, src)

	case *ast.String:
		buf.WriteString(escapeHTML(string(n.Value)))

	case *ast.Emphasis:
		writeEmphasis(buf, n, src, depth)

	case *ast.CodeSpan:
		writeCodeSpan(buf, n, src)

	case *ast.Link:
		writeLink(buf, n, src, depth)

	case *ast.Image:
		writeImage(buf, n, src, depth)

	case *ast.AutoLink:
		writeAutoLink(buf, n, src)

	case *ast.RawHTML:
		writeRawHTML(buf, n, src)

	case *extast.Strikethrough:
		buf.WriteString("<s>")
		writeChildren(buf, n, src, depth)
		buf.WriteString("</s>")

	case *extast.Table:
		renderTable(buf, n, src, depth)

	case *extast.TableHeader, *extast.TableRow, *extast.TableCell:
		// handled by renderTable

	default:
		// For unknown nodes, recurse into children
		writeChildren(buf, node, src, depth)
	}
}

func writeChildren(buf *bytes.Buffer, node ast.Node, src []byte, depth int) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		walkNode(buf, child, src, depth)
	}
}

func writeParagraph(buf *bytes.Buffer, node *ast.Paragraph, src []byte, depth int) {
	var inner bytes.Buffer
	writeChildren(&inner, node, src, depth)
	content := inner.String()
	if content == "" {
		return
	}
	ensureDoubleNewline(buf)
	buf.WriteString(content)
}

// uppercaseTextOnly uppercases only the text characters in an HTML string,
// leaving tag names and attribute values untouched.
func uppercaseTextOnly(html string) string {
	var buf strings.Builder
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
			buf.WriteRune(r)
		} else if r == '>' {
			inTag = false
			buf.WriteRune(r)
		} else if inTag {
			buf.WriteRune(r)
		} else {
			buf.WriteRune(unicode.ToUpper(r))
		}
	}
	return buf.String()
}

func writeHeading(buf *bytes.Buffer, node *ast.Heading, src []byte, depth int) {
	var inner bytes.Buffer
	writeChildren(&inner, node, src, depth)
	content := inner.String()

	ensureDoubleNewline(buf)

	if node.Level == 1 {
		buf.WriteString("<b>")
		buf.WriteString(uppercaseTextOnly(content))
		buf.WriteString("</b>")
	} else {
		buf.WriteString("<b>")
		buf.WriteString(content)
		buf.WriteString("</b>")
	}
}

func writeFencedCode(buf *bytes.Buffer, node *ast.FencedCodeBlock, src []byte) {
	ensureDoubleNewline(buf)

	lang := string(node.Language(src))
	if lang != "" {
		buf.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">", escapeHTML(lang)))
	} else {
		buf.WriteString("<pre><code>")
	}

	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		buf.WriteString(escapeHTML(string(line.Value(src))))
	}

	buf.WriteString("</code></pre>")
}

func writeCodeBlock(buf *bytes.Buffer, node *ast.CodeBlock, src []byte) {
	ensureDoubleNewline(buf)

	buf.WriteString("<pre><code>")
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		buf.WriteString(escapeHTML(string(line.Value(src))))
	}
	buf.WriteString("</code></pre>")
}

func writeBlockquote(buf *bytes.Buffer, node *ast.Blockquote, src []byte, depth int) {
	ensureDoubleNewline(buf)

	buf.WriteString("<blockquote>")
	// Blockquote contains paragraphs — render their inner content directly
	var inner bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if p, ok := child.(*ast.Paragraph); ok {
			writeChildren(&inner, p, src, depth)
		} else {
			walkNode(&inner, child, src, depth)
		}
	}
	buf.WriteString(strings.TrimSpace(inner.String()))
	buf.WriteString("</blockquote>")
}

func writeList(buf *bytes.Buffer, node *ast.List, src []byte, depth int) {
	if depth == 0 {
		ensureDoubleNewline(buf)
	}

	isFirst := true
	itemNum := node.Start
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if li, ok := child.(*ast.ListItem); ok {
			if !isFirst {
				buf.WriteString("\n")
			}
			isFirst = false

			indent := strings.Repeat("  ", depth)
			buf.WriteString(indent)

			if node.IsOrdered() {
				buf.WriteString(fmt.Sprintf("%d. ", itemNum))
				itemNum++
			} else {
				buf.WriteString("• ")
			}

			writeListItemContent(buf, li, src, depth+1)
		}
	}
}

func writeListItemContent(buf *bytes.Buffer, node *ast.ListItem, src []byte, depth int) {
	// A list item may have a paragraph (tight lists have TextBlock) and sub-lists
	firstBlock := true
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.TextBlock:
			if !firstBlock {
				buf.WriteString("\n")
			}
			writeChildren(buf, c, src, depth)
			firstBlock = false
		case *ast.Paragraph:
			if !firstBlock {
				buf.WriteString("\n")
			}
			writeChildren(buf, c, src, depth)
			firstBlock = false
		case *ast.List:
			buf.WriteString("\n")
			writeList(buf, c, src, depth)
			firstBlock = false
		default:
			walkNode(buf, child, src, depth)
		}
	}
}

func writeThematicBreak(buf *bytes.Buffer, node *ast.ThematicBreak) {
	ensureDoubleNewline(buf)
	buf.WriteString("————————————")
	if node.NextSibling() != nil {
		buf.WriteString("\n\n")
	}
}

func writeText(buf *bytes.Buffer, node *ast.Text, src []byte) {
	seg := node.Segment
	val := seg.Value(src)
	buf.WriteString(escapeHTML(string(val)))
	if node.SoftLineBreak() {
		buf.WriteString("\n")
	} else if node.HardLineBreak() {
		buf.WriteString("\n")
	}
}

func writeEmphasis(buf *bytes.Buffer, node *ast.Emphasis, src []byte, depth int) {
	if node.Level == 2 {
		buf.WriteString("<b>")
		writeChildren(buf, node, src, depth)
		buf.WriteString("</b>")
	} else {
		buf.WriteString("<i>")
		writeChildren(buf, node, src, depth)
		buf.WriteString("</i>")
	}
}

func writeCodeSpan(buf *bytes.Buffer, node *ast.CodeSpan, src []byte) {
	buf.WriteString("<code>")
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			seg := t.Segment
			buf.WriteString(escapeHTML(string(seg.Value(src))))
		} else if s, ok := child.(*ast.String); ok {
			buf.WriteString(escapeHTML(string(s.Value)))
		}
	}
	buf.WriteString("</code>")
}

func writeLink(buf *bytes.Buffer, node *ast.Link, src []byte, depth int) {
	href := escapeHTML(string(node.Destination))
	buf.WriteString(fmt.Sprintf(`<a href="%s">`, href))
	writeChildren(buf, node, src, depth)
	buf.WriteString("</a>")
}

func writeImage(buf *bytes.Buffer, node *ast.Image, src []byte, depth int) {
	href := escapeHTML(string(node.Destination))
	buf.WriteString(fmt.Sprintf(`<a href="%s">`, href))
	writeChildren(buf, node, src, depth)
	buf.WriteString("</a>")
}

func writeAutoLink(buf *bytes.Buffer, node *ast.AutoLink, src []byte) {
	url := escapeHTML(string(node.URL(src)))
	buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, url, url))
}

func writeRawHTML(buf *bytes.Buffer, node *ast.RawHTML, src []byte) {
	// Escape raw HTML so it renders as literal text in Telegram
	segs := node.Segments
	for i := 0; i < segs.Len(); i++ {
		seg := segs.At(i)
		buf.WriteString(escapeHTML(string(seg.Value(src))))
	}
}

// collectTableRow extracts the text content of each cell in a row node.
func collectTableRow(row ast.Node, src []byte) []string {
	var cells []string
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		var cellBuf bytes.Buffer
		writeChildren(&cellBuf, cell, src, 0)
		cells = append(cells, strings.TrimSpace(cellBuf.String()))
	}
	return cells
}

// renderTable renders an extast.Table as monospace text inside <pre> tags.
func renderTable(buf *bytes.Buffer, node *extast.Table, src []byte, depth int) {
	var rows [][]string

	// Collect all rows: header first, then body rows.
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch r := child.(type) {
		case *extast.TableHeader:
			// TableHeader holds TableCell children directly.
			if r.FirstChild() != nil {
				if _, ok := r.FirstChild().(*extast.TableCell); ok {
					rows = append(rows, collectTableRow(r, src))
				}
			}
		case *extast.TableRow:
			rows = append(rows, collectTableRow(r, src))
		}
	}

	if len(rows) == 0 {
		return
	}

	// Determine number of columns.
	numCols := 0
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Calculate max width per column.
	colWidths := make([]int, numCols)
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	ensureDoubleNewline(buf)
	buf.WriteString("<pre>")

	for rowIdx, row := range rows {
		// Build the row line.
		for colIdx := 0; colIdx < numCols; colIdx++ {
			cell := ""
			if colIdx < len(row) {
				cell = row[colIdx]
			}
			if colIdx < numCols-1 {
				// Pad cell to column width for all but the last column.
				padded := cell + strings.Repeat(" ", colWidths[colIdx]-len(cell))
				buf.WriteString(padded)
				buf.WriteString(" | ")
			} else {
				// Last column: no trailing padding.
				buf.WriteString(cell)
			}
		}

		buf.WriteString("\n")

		// After the header row (index 0), write the separator line.
		if rowIdx == 0 {
			for colIdx := 0; colIdx < numCols; colIdx++ {
				buf.WriteString(strings.Repeat("-", colWidths[colIdx]+1))
				if colIdx < numCols-1 {
					buf.WriteString("+")
				}
			}
			buf.WriteString("\n")
		}
	}

	// Trim the trailing newline inside the <pre> block, then close it.
	preStart := strings.LastIndex(buf.String(), "<pre>")
	trimmed := strings.TrimRight(buf.String()[preStart+5:], "\n")
	buf.Truncate(preStart)
	buf.WriteString("<pre>")
	buf.WriteString(trimmed)
	buf.WriteString("</pre>")

	if node.NextSibling() != nil {
		buf.WriteString("\n\n")
	}
}
