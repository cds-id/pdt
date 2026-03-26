/**
 * Converts Jira wiki markup to standard markdown.
 */
export function jiraToMarkdown(text: string): string {
  if (!text) return ''

  let result = text

  // Code blocks: {code:language}...{code} or {code}...{code}
  result = result.replace(/\{code(?::(\w+))?\}([\s\S]*?)\{code\}/g, (_match, lang, code) => {
    const language = lang || ''
    return `\`\`\`${language}\n${code.trim()}\n\`\`\``
  })

  // Inline code: {{text}}
  result = result.replace(/\{\{(.+?)\}\}/g, '`$1`')

  // Headings: h1. through h6. at line start
  result = result.replace(/^h([1-6])\.\s*(.*)$/gm, (_match, level, content) => {
    return '#'.repeat(Number(level)) + ' ' + content
  })

  // Bold: *text* → **text** (but not inside words or URLs)
  result = result.replace(/(?<!\w)\*([^\s*](?:[^*]*[^\s*])?)\*(?!\w)/g, '**$1**')

  // Italic: _text_ → *text* (but not inside words or URLs)
  result = result.replace(/(?<!\w)_([^\s_](?:[^_]*[^\s_])?)_(?!\w)/g, '*$1*')

  // Ordered list: # item → 1. item (at line start, only single #)
  result = result.replace(/^#\s+(.*)$/gm, '1. $1')

  // Links: [text|url] → [text](url)
  result = result.replace(/\[([^|\]]+)\|([^\]]+)\]/g, '[$1]($2)')

  // Links: [url] → [url](url) (bare URLs in brackets)
  result = result.replace(/\[(https?:\/\/[^\]]+)\]/g, '[$1]($1)')

  // Blockquote: {quote}...{quote}
  result = result.replace(/\{quote\}([\s\S]*?)\{quote\}/g, (_match, content) => {
    return content.trim().split('\n').map((line: string) => `> ${line}`).join('\n')
  })

  // Panel: {panel}...{panel} → blockquote
  result = result.replace(/\{panel(?::[^}]*)?\}([\s\S]*?)\{panel\}/g, (_match, content) => {
    return content.trim().split('\n').map((line: string) => `> ${line}`).join('\n')
  })

  // Strip color tags: {color:#hex}text{color} → text
  result = result.replace(/\{color(?::[^}]*)?\}([\s\S]*?)\{color\}/g, '$1')

  // Strikethrough: -text- → ~~text~~
  result = result.replace(/(?<!\w)-([^\s-](?:[^-]*[^\s-])?)-(?!\w)/g, '~~$1~~')

  // Jira table: ||header|| → | header |
  result = result.replace(/^\|\|(.+)\|\|$/gm, (_match, content) => {
    const cells = content.split('||').map((c: string) => c.trim())
    const header = '| ' + cells.join(' | ') + ' |'
    const separator = '| ' + cells.map(() => '---').join(' | ') + ' |'
    return header + '\n' + separator
  })

  // Jira table rows: |cell|cell| → | cell | cell |
  result = result.replace(/^\|([^|].*)\|$/gm, (_match, content) => {
    const cells = content.split('|').map((c: string) => c.trim())
    return '| ' + cells.join(' | ') + ' |'
  })

  return result
}
