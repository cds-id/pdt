# Telegram Response Formatting Design

## Problem

The Telegram bot receives standard Markdown from the LLM (headings, tables, `**bold**`, lists) but Telegram MarkdownV2 has different syntax and requires escaping 18 special characters. This causes parse failures, falling back to plain text.

Additionally, sessions are shared between Telegram and webchat. Each channel needs its own formatting — the conversion must happen at the transport layer, not the orchestrator.

## Decision

Replace MarkdownV2 with **Telegram HTML** mode using a post-processing Markdown-to-HTML converter. HTML only requires escaping 3 characters (`<`, `>`, `&`) vs 18 for MarkdownV2, making it far more reliable for dynamic LLM content.

## Architecture

```
LLM -> standard Markdown -> orchestrator -> StreamWriter
                                               |
                                    formatter.ToTelegramHTML(content)
                                               |
                                    msg.ParseMode = "HTML"
                                    bot.Send(msg)
```

- Conversion happens in `stream_writer.go` (transport layer)
- Orchestrator and LLM continue outputting standard Markdown
- Webchat renders standard Markdown natively (no change)
- No changes to LLM prompts or session sharing

## Conversion Rules

| Standard Markdown | Telegram HTML Output |
|---|---|
| `# Heading` | `<b>HEADING</b>` (uppercase for h1) |
| `## Subheading` | `<b>Subheading</b>` |
| `### H3+` | `<b>H3+</b>` |
| `**bold**` | `<b>bold</b>` |
| `*italic*` / `_italic_` | `<i>italic</i>` |
| `~~strikethrough~~` | `<s>strikethrough</s>` |
| `` `inline code` `` | `<code>inline code</code>` |
| ` ```lang\ncode\n``` ` | `<pre><code class="language-lang">code</code></pre>` |
| `[text](url)` | `<a href="url">text</a>` |
| `- item` / `* item` | `• item` (bullet character) |
| `1. item` | `1. item` (plain numbered) |
| Nested lists | Indented with spaces |
| `> blockquote` | `<blockquote>text</blockquote>` |
| Tables | `<pre>` monospace alignment |
| `---` horizontal rule | `————————————` |
| `![alt](url)` | `<a href="url">alt</a>` |

**Escaping:** Only `<`, `>`, `&` escaped to `&lt;`, `&gt;`, `&amp;` on text nodes before wrapping in tags.

**Fallback:** If HTML send fails, strip all tags and send as plain text.

## Implementation Scope

### Files to change

1. **`backend/internal/services/telegram/stream_writer.go`**
   - Replace `escapeMarkdownV2()` calls with `formatter.ToTelegramHTML()`
   - Switch `ParseMode` from `"MarkdownV2"` to `"HTML"`
   - Update plain-text fallback to strip HTML tags
   - Remove `escapeMarkdownV2()` function

2. **New: `backend/internal/services/telegram/formatter/telegram_html.go`**
   - Uses `goldmark` to parse Markdown AST
   - Custom renderer walks AST and outputs Telegram HTML per conversion rules
   - Pure function: `func ToTelegramHTML(markdown string) string`

3. **New: `backend/internal/services/telegram/formatter/telegram_html_test.go`**
   - Table-driven tests for each conversion rule
   - Tests: headings, bold, italic, code blocks, lists, tables, links, images, blockquotes, horizontal rules, nested formatting, HTML escaping

### Files NOT changed

- `handler.go` — outbox messages and commands stay plain text
- Orchestrator, agents, LLM prompts — no changes
- Webchat — no changes

### Dependency

- `github.com/yuin/goldmark` — mature Go Markdown parser with extensible AST

## Risk Mitigation

- Plain-text fallback remains for edge cases where HTML parsing fails
- Converter is a pure function, easy to unit test in isolation
- No behavioral change to orchestrator or session sharing
- Chunking logic (`splitMessage`) unchanged
