const FRONTMATTER_RE = /---\n([\s\S]*?)\n---/

export function extractTriggerFrontmatter(text: string): Record<string, string> {
  const match = text.match(FRONTMATTER_RE)
  if (!match?.[1]) return {}
  return match[1]
    .split('\n')
    .reduce<Record<string, string>>((acc, line) => {
      const idx = line.indexOf(':')
      if (idx < 0) return acc
      const key = line.slice(0, idx).trim()
      const value = line.slice(idx + 1).trim()
      if (key) acc[key] = value
      return acc
    }, {})
}

export function extractTriggerBody(text: string): string {
  const match = text.match(FRONTMATTER_RE)
  if (!match) return text.trim()
  const end = match.index! + match[0].length
  return text.slice(end).trim()
}
