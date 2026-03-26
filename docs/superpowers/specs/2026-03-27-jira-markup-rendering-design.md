# Jira Wiki Markup Rendering

## Overview

Render Jira descriptions and comments with proper formatting on the card detail page. Currently they display as raw text with visible wiki markup (`*bold*`, `{code}`, `h2.`, etc.). Convert Jira wiki markup to markdown on the frontend and render with `MessageResponse` (Streamdown).

## Current State

- Description: rendered as `whitespace-pre-wrap` plain text in `JiraCardDetailPage.tsx:103`
- Comments: rendered as `whitespace-pre-wrap` plain text in `JiraCardDetailPage.tsx:207-208`
- Both show raw Jira wiki markup to the user

## Solution

### Converter: `jiraToMarkdown(text: string): string`

A pure function that converts Jira wiki markup to standard markdown using regex replacements.

Supported conversions:

| Jira Wiki | Markdown |
|-----------|----------|
| `*bold*` | `**bold**` |
| `_italic_` | `*italic*` |
| `h1.` through `h6.` (at line start) | `#` through `######` |
| `# item` (ordered list at line start) | `1. item` |
| `- item` (unordered list) | `- item` (passthrough) |
| `{code}...{code}` | ` ``` ``` ` |
| `{code:language}...{code}` | ` ```language ``` ` |
| `{{monospace}}` | `` `monospace` `` |
| `[text\|url]` | `[text](url)` |
| `[url]` | `[url](url)` |
| `{quote}...{quote}` | `> ...` |
| `{panel}...{panel}` | `> ...` (same as quote) |
| `{color}...{color}` | strip color tags, keep content |
| `\n` | preserved |

### Rendering

Use `MessageResponse` from `components/ai-elements/message.tsx` which renders markdown via Streamdown with Shiki syntax highlighting, math, and mermaid support.

## Files

### Create
- `frontend/src/lib/jira-markup.ts` — `jiraToMarkdown()` converter function

### Modify
- `frontend/src/presentation/pages/JiraCardDetailPage.tsx` — import converter + `MessageResponse`, use them for description and comments sections

## Success Criteria

- Jira descriptions render with proper headings, lists, bold, italic, code blocks
- Jira comments render with the same formatting
- Code blocks in descriptions/comments get Shiki syntax highlighting
- Raw wiki markup is no longer visible to the user
- Plain text descriptions (no wiki markup) still render correctly
